package pkglib

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/docker/docker/api/types"
	registry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type buildOpts struct {
	skipBuild        bool
	force            bool
	pull             bool
	ignoreCache      bool
	push             bool
	release          string
	manifest         bool
	targetDocker     bool
	cacheDir         string
	cacheProvider    lktspec.CacheProvider
	platforms        []imagespec.Platform
	builders         map[string]string
	runner           dockerRunner
	writer           io.Writer
	builderImage     string
	builderRestart   bool
	sbomScan         bool
	sbomScannerImage string
	dockerfile       string
	buildArgs        []string
	progress         string
}

// BuildOpt allows callers to specify options to Build
type BuildOpt func(bo *buildOpts) error

// WithBuildSkip skips the actual build and only pushes/releases (if configured)
func WithBuildSkip() BuildOpt {
	return func(bo *buildOpts) error {
		bo.skipBuild = true
		return nil
	}
}

// WithBuildForce forces a build even if an image already exists
func WithBuildForce() BuildOpt {
	return func(bo *buildOpts) error {
		bo.force = true
		return nil
	}
}

// WithBuildPull pull down the image to cache if it already exists in registry
func WithBuildPull() BuildOpt {
	return func(bo *buildOpts) error {
		bo.pull = true
		return nil
	}
}

// WithBuildPush pushes the result of the build to the registry
func WithBuildPush() BuildOpt {
	return func(bo *buildOpts) error {
		bo.push = true
		return nil
	}
}

// WithBuildManifest creates a multi-arch manifest for the image
func WithBuildManifest() BuildOpt {
	return func(bo *buildOpts) error {
		bo.manifest = true
		return nil
	}
}

// WithRelease releases as the given version after push
func WithRelease(r string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.release = r
		return nil
	}
}

// WithBuildTargetDockerCache put the build target in the docker cache instead of the default linuxkit cache
func WithBuildTargetDockerCache() BuildOpt {
	return func(bo *buildOpts) error {
		bo.targetDocker = true
		bo.pull = true // if we are to load it into docker, it must be in local cache
		return nil
	}
}

// WithBuildCacheDir provide a build cache directory to use
func WithBuildCacheDir(dir string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.cacheDir = dir
		return nil
	}
}

// WithBuildPlatforms which platforms to build for
func WithBuildPlatforms(platforms ...imagespec.Platform) BuildOpt {
	return func(bo *buildOpts) error {
		bo.platforms = platforms
		return nil
	}
}

// WithBuildBuilders which builders, as named contexts per platform, to use
func WithBuildBuilders(builders map[string]string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.builders = builders
		return nil
	}
}

// WithBuildDocker provides a docker runner to use. If nil, defaults to the current platform
func WithBuildDocker(runner dockerRunner) BuildOpt {
	return func(bo *buildOpts) error {
		bo.runner = runner
		return nil
	}
}

// WithBuildCacheProvider provides a cacheProvider to use. If nil, defaults to the one shipped with linuxkit
func WithBuildCacheProvider(c lktspec.CacheProvider) BuildOpt {
	return func(bo *buildOpts) error {
		bo.cacheProvider = c
		return nil
	}
}

// WithBuildOutputWriter set the output writer for messages. If nil, defaults to stdout
func WithBuildOutputWriter(w io.Writer) BuildOpt {
	return func(bo *buildOpts) error {
		bo.writer = w
		return nil
	}
}

// WithBuildBuilderImage set the builder container image to use.
func WithBuildBuilderImage(image string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.builderImage = image
		return nil
	}
}

// WithBuildBuilderRestart restart the builder container even if it already is running with the correct image version
func WithBuildBuilderRestart(restart bool) BuildOpt {
	return func(bo *buildOpts) error {
		bo.builderRestart = restart
		return nil
	}
}

// WithBuildIgnoreCache when building an image, do not look in local cache for dependent images
func WithBuildIgnoreCache() BuildOpt {
	return func(bo *buildOpts) error {
		bo.ignoreCache = true
		return nil
	}
}

// WithBuildSbomScanner when building an image, scan using the provided scanner image; if blank, uses the default
func WithBuildSbomScanner(scanner string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.sbomScan = true
		bo.sbomScannerImage = scanner
		return nil
	}
}

// WithDockerfile which dockerfile to use when building the package
func WithDockerfile(dockerfile string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.dockerfile = dockerfile
		return nil
	}
}

// WithBuildArgs add build args to use when building the package
func WithBuildArgs(args []string) BuildOpt {
	return func(bo *buildOpts) error {
		// we copy the contents, rather than the reference to the slice, to be safe
		bo.buildArgs = make([]string, len(args))
		copy(bo.buildArgs, args)
		return nil
	}
}

// WithProgress which progress type to show
func WithProgress(progress string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.progress = progress
		return nil
	}
}

// Build builds the package
func (p Pkg) Build(bos ...BuildOpt) error {
	var bo buildOpts
	var ctx = context.TODO()
	for _, fn := range bos {
		if err := fn(&bo); err != nil {
			return err
		}
	}

	writer := bo.writer
	if writer == nil {
		writer = os.Stdout
	}

	arch := runtime.GOARCH
	ref, err := reference.Parse(p.FullTag())
	if err != nil {
		return fmt.Errorf("could not resolve references for image %s: %v", p.Tag(), err)
	}

	if err := p.cleanForBuild(); err != nil {
		return err
	}

	// validate the Dockerfile before bothing to move ahead, because this func call is public, so someone could
	// pass something to it as a library call. We also check in the build function, to avoid multiple loops each with an error.

	// if the dockerfile override was not set in the build options, i.e. it is empty, use the one from the package,
	// which never should be empty. We set it onto the buildOpts, because that is what we use to pass it around to lower-level
	// funcs.
	if bo.dockerfile == "" {
		bo.dockerfile = p.dockerfile
	}
	if strings.Contains(bo.dockerfile, "..") {
		return fmt.Errorf("cannot expand beyond root of context for dockerfile %s", bo.dockerfile)
	}
	if _, err := os.Stat(filepath.Join(p.path, bo.dockerfile)); err != nil {
		return fmt.Errorf("dockerfile %s does not exist or cannot be read in context %s", bo.dockerfile, p.path)
	}

	// did we have the build cache dir provided?
	if bo.cacheDir == "" {
		return errors.New("must provide linuxkit build cache directory")
	}

	if p.git != nil && bo.push && bo.release == "" {
		r, err := p.git.commitTag("HEAD")
		if err != nil {
			return err
		}
		bo.release = r
	}

	if bo.release != "" && !bo.push {
		return fmt.Errorf("cannot release %q if not pushing", bo.release)
	}

	d := bo.runner
	if d == nil {
		d = newDockerRunner(p.cache)
	}

	c := bo.cacheProvider
	if c == nil {
		c, err = cache.NewProvider(bo.cacheDir)
		if err != nil {
			return err
		}
	}

	if err := d.contextSupportCheck(); err != nil {
		return fmt.Errorf("contexts not supported, check docker version: %v", err)
	}

	var (
		platformsToBuild []imagespec.Platform
		// imageInLocalCache flags if we had at least one image in local cache. If we had at least one,
		// and push was requested, we will try to push.
		imageInLocalCache bool
	)
	switch {
	case bo.force && bo.skipBuild:
		return errors.New("cannot force build and skip build")
	case bo.force:
		// force local build
		platformsToBuild = bo.platforms
	case bo.skipBuild:
		// do not build anything if we explicitly did skipBuild
		platformsToBuild = nil
	default:
		// check local cache, fallback to check registry / pull image from registry, fallback to build
		fmt.Fprintf(writer, "checking for %s in local cache...\n", ref)
		for _, platform := range bo.platforms {
			exists, err := c.ImageInCache(&ref, "", platform.Architecture)
			switch {
			case err == nil && exists:
				fmt.Fprintf(writer, "found %s in local cache, skipping build\n", ref)
				imageInLocalCache = true
				continue
			case bo.pull:
				// need to pull the image from the registry, else build
				fmt.Fprintf(writer, "%s %s not found in local cache, trying to pull\n", ref, platform.Architecture)
				if _, err := c.ImagePull(&ref, "", platform.Architecture, false); err == nil {
					fmt.Fprintf(writer, "%s pulled\n", ref)
					// successfully pulled, no need to build, continue with next platform
					continue
				}
				fmt.Fprintf(writer, "%s not found, will build: %s\n", ref, err)
				platformsToBuild = append(platformsToBuild, platform)
			default:
				// do not pull, just check if it exists in a registry
				fmt.Fprintf(writer, "%s %s not found in local cache, checking registry\n", ref, platform.Architecture)
				exists, err := c.ImageInRegistry(&ref, "", platform.Architecture)
				if err != nil {
					return fmt.Errorf("error checking remote registry for %s: %v", ref, err)
				}

				if exists {
					fmt.Fprintf(writer, "%s %s found on registry\n", ref, platform.Architecture)
					continue
				}
				fmt.Fprintf(writer, "%s %s not found, will build\n", ref, platform.Architecture)
				platformsToBuild = append(platformsToBuild, platform)
			}
		}
	}

	if len(platformsToBuild) > 0 {
		var arches []string
		for _, platform := range platformsToBuild {
			arches = append(arches, platform.Architecture)
		}
		fmt.Fprintf(writer, "building %s for arches: %s\n", ref, strings.Join(arches, ","))
		var (
			imageBuildOpts = types.ImageBuildOptions{
				Labels:    map[string]string{},
				BuildArgs: map[string]*string{},
			}
			descs []registry.Descriptor
		)

		// args that we use:
		//   labels map[string]string
		//   network string
		//   build-arg []string

		if p.git != nil && p.gitRepo != "" {
			imageBuildOpts.Labels["org.opencontainers.image.source"] = p.gitRepo
		}
		var (
			gitCommit    string
			goPkgVersion string
		)
		if p.git != nil {
			if !p.dirty {
				gitCommit, err = p.git.commitHash("HEAD")
				if err != nil {
					return err
				}
				imageBuildOpts.Labels["org.opencontainers.image.revision"] = gitCommit
			}
			// get the go version or pseudo-version
			goPkgVersion, err = p.git.goPkgVersion()
			if err != nil {
				return err
			}
		}

		imageBuildOpts.NetworkMode = "default"
		if !p.network {
			imageBuildOpts.NetworkMode = "none"
		}

		if p.config != nil {
			b, err := json.Marshal(*p.config)
			if err != nil {
				return err
			}
			imageBuildOpts.Labels["org.mobyproject.config"] = string(b)
		}

		imageBuildOpts.Labels["org.mobyproject.linuxkit.version"] = version.Version
		imageBuildOpts.Labels["org.mobyproject.linuxkit.revision"] = version.GitCommit

		// add build args from the build.yml file
		if p.buildArgs != nil {
			for _, buildArg := range *p.buildArgs {
				parts := strings.SplitN(buildArg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", buildArg)
				}
				imageBuildOpts.BuildArgs[parts[0]] = &parts[1]
			}
		}
		// add build args from other files
		for _, buildArg := range bo.buildArgs {
			parts := strings.SplitN(buildArg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", buildArg)
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			imageBuildOpts.BuildArgs[key] = &val
		}

		// add in information about the build process that might be useful
		if _, ok := imageBuildOpts.BuildArgs["SOURCE"]; !ok && p.gitRepo != "" {
			imageBuildOpts.BuildArgs["SOURCE"] = &p.gitRepo
		}
		if _, ok := imageBuildOpts.BuildArgs["REVISION"]; !ok && gitCommit != "" {
			imageBuildOpts.BuildArgs["REVISION"] = &gitCommit
		}
		if _, ok := imageBuildOpts.BuildArgs["GOPKGVERSION"]; !ok && goPkgVersion != "" {
			imageBuildOpts.BuildArgs["GOPKGVERSION"] = &goPkgVersion
		}
		if _, ok := imageBuildOpts.BuildArgs["PKG_HASH"]; !ok && p.Hash() != "" {
			ret := p.Hash()
			imageBuildOpts.BuildArgs["PKG_HASH"] = &ret
		}
		if _, ok := imageBuildOpts.BuildArgs["PKG_IMAGE"]; !ok && p.Image() != "" {
			ret := p.Image()
			imageBuildOpts.BuildArgs["PKG_IMAGE"] = &ret
		}

		// build for each arch and save in the linuxkit cache
		for _, platform := range platformsToBuild {
			builtDescs, err := p.buildArch(ctx, d, c, bo.builderImage, platform.Architecture, bo.builderRestart, writer, bo, imageBuildOpts)
			if err != nil {
				return fmt.Errorf("error building for arch %s: %v", platform.Architecture, err)
			}
			if len(builtDescs) == 0 {
				return fmt.Errorf("no valid descriptor returned for image for arch %s", platform.Architecture)
			}
			descs = append(descs, builtDescs...)
		}

		// after build is done:
		// - create multi-arch index
		// - potentially push
		// - potentially load into docker
		// - potentially create a release, including push and load into docker

		// create a multi-arch index
		if _, err := c.IndexWrite(&ref, descs...); err != nil {
			return err
		}
	}

	// get descriptor for root of manifest
	desc, err := c.FindDescriptor(&ref)
	if err != nil {
		return err
	}

	// if requested docker, load the image up
	// we will store images with arch suffix, i.e. -amd64
	// if one of the arch equals with system, we will add tag without suffix
	if bo.targetDocker {
		for _, platform := range bo.platforms {
			ref, err := reference.Parse(p.FullTag())
			if err != nil {
				return err
			}
			cacheSource := c.NewSource(&ref, platform.Architecture, desc)
			reader, err := cacheSource.V1TarReader(fmt.Sprintf("%s-%s", p.FullTag(), platform.Architecture))
			if err != nil {
				return fmt.Errorf("unable to get reader from cache: %v", err)
			}
			if err := d.load(reader); err != nil {
				return err
			}
			if platform.Architecture == arch {
				err = d.tag(fmt.Sprintf("%s-%s", p.FullTag(), platform.Architecture), p.FullTag())
				if err != nil {
					return err
				}
			}
		}
	}

	if !bo.push {
		fmt.Fprintf(writer, "Build complete, not pushing, all done.\n")
		return nil
	}

	// we only will push if one of these is true:
	// - we had at least one platform to build
	// - we found an image in local cache
	// if neither is true, there is nothing to push

	if len(platformsToBuild) == 0 {
		// if we did not yet find the image in local cache,
		// check, in case we have it and would need to push.
		// If we did not build it because we were not requested to do so,
		// then we might not know we have it in local cache.
		if !imageInLocalCache {
			// we need this to know whether or not we might push
			for _, platform := range bo.platforms {
				exists, err := c.ImageInCache(&ref, "", platform.Architecture)
				if err == nil && exists {
					imageInLocalCache = true
					break
				}
			}
		}
		if !imageInLocalCache {
			fmt.Fprintf(writer, "No new platforms to push, skipping.\n")
			return nil
		}
	}

	if p.dirty {
		return fmt.Errorf("build complete, refusing to push dirty package")
	}

	// push the manifest
	if err := c.Push(p.FullTag(), "", bo.manifest, true); err != nil {
		return err
	}

	if bo.release == "" {
		fmt.Fprintf(writer, "Build and push complete, not releasing, all done.\n")
		return nil
	}

	relTag, err := p.ReleaseTag(bo.release)
	if err != nil {
		return err
	}
	fullRelTag := util.ReferenceExpand(relTag)

	ref, err = reference.Parse(fullRelTag)
	if err != nil {
		return err
	}
	if _, err := c.DescriptorWrite(&ref, *desc); err != nil {
		return err
	}
	if err := c.Push(fullRelTag, "", bo.manifest, true); err != nil {
		return err
	}

	// tag in docker, if requested
	// will tag all images with arch suffix
	// if one of the arch equals with system will add tag without suffix
	if bo.targetDocker {
		for _, platform := range bo.platforms {
			if err := d.tag(fmt.Sprintf("%s-%s", p.FullTag(), platform.Architecture), fmt.Sprintf("%s-%s", fullRelTag, platform.Architecture)); err != nil {
				return err
			}
			if platform.Architecture == arch {
				if err := d.tag(fmt.Sprintf("%s-%s", p.FullTag(), platform.Architecture), fullRelTag); err != nil {
					return err
				}
			}
		}
	}

	fmt.Fprintf(writer, "Build, push and release of %q complete, all done.\n", bo.release)

	return nil
}

// buildArch builds the package for a single arch, and loads the result in the cache provided in the argument.
// Unless force is set, it will check the cache for the image first, then on the registry, and if it exists, it will not build it.
// The image will be saved in the cache with the provided package name and tag, with the architecture appended
// as a suffix, i.e. "myimage:abc-amd64" or "myimage:abc-arm64".
// It returns a list of individual descriptors for the images built, which can be used to create an index.
// These descriptors are not of the index pointed to by "myimage:abc-amd64", but rather the underlying manifests
// in that index.
// The final result then is as follows:
// A - layers, saved in cache as is
// B - config, saved in cache as is
// C - manifest, saved in cache as is, referenced by the index (E), and returned as a descriptor
// D - attestations (if any), saved in cache as is, referenced by the index (E), and returned as a descriptor
// E - index, saved in cache as is, stored in cache as tag "image:tag-arch", *not* returned as a descriptor
func (p Pkg) buildArch(ctx context.Context, d dockerRunner, c lktspec.CacheProvider, builderImage, arch string, restart bool, writer io.Writer, bo buildOpts, imageBuildOpts types.ImageBuildOptions) ([]registry.Descriptor, error) {
	var (
		tagArch   string
		tag       = p.FullTag()
		indexDesc []registry.Descriptor
	)
	tagArch = tag + "-" + arch
	fmt.Fprintf(writer, "Building for arch %s as %s\n", arch, tagArch)

	if !bo.force {
		ref, err := reference.Parse(p.FullTag())
		if err != nil {
			return nil, fmt.Errorf("could not resolve references for image %s: %v", p.Tag(), err)
		}
		if _, err := c.ImagePull(&ref, "", arch, false); err == nil {
			fmt.Fprintf(writer, "image already found %s for arch %s", ref, arch)
			desc, err := c.FindDescriptor(&ref)
			if err != nil {
				return nil, fmt.Errorf("could not find root descriptor for %s: %v", ref, err)
			}
			return []registry.Descriptor{*desc}, nil
		}
		fmt.Fprintf(writer, "No image pulled for arch %s, continuing with build\n", arch)
	}

	if err := p.dockerDepends.Do(d); err != nil {
		return nil, err
	}

	// find the desired builder
	builderName := getBuilderForPlatform(arch, bo.builders)

	// set the target
	var (
		stdout       io.WriteCloser
		eg           errgroup.Group
		stdoutCloser = func() {
			if stdout != nil {
				stdout.Close()
			}
		}
	)

	// we are writing to local, so we need to catch the tar output stream and place the right files in the right place
	piper, pipew := io.Pipe()
	stdout = pipew

	eg.Go(func() error {
		d, err := c.ImageLoad(piper)
		// send the error down the channel
		if err != nil {
			fmt.Fprintf(stdout, "cache.ImageLoad goroutine ended with error: %v\n", err)
		} else {
			indexDesc = d
		}
		piper.Close()
		return err
	})

	buildCtx := &buildCtx{sources: p.sources}
	platform := fmt.Sprintf("linux/%s", arch)
	// if we were told to ignore cached dependent images, pass it a nil cache so it cannot read anything
	passCache := c
	if bo.ignoreCache {
		passCache = nil
	}

	if err := d.build(ctx, tagArch, p.path, bo.dockerfile, builderName, builderImage, platform, restart, passCache, buildCtx.Reader(), stdout, bo.sbomScan, bo.sbomScannerImage, bo.progress, imageBuildOpts); err != nil {
		stdoutCloser()
		if strings.Contains(err.Error(), "executor failed running [/dev/.buildkit_qemu_emulator") {
			return nil, fmt.Errorf("buildkit was unable to emulate %s. check binfmt has been set up and works for this platform: %v", platform, err)
		}
		return nil, err
	}
	stdoutCloser()

	// wait for the processor to finish
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// find the child manifests
	// how many index descriptors did we get?
	switch len(indexDesc) {
	case 0:
		return nil, fmt.Errorf("no index descriptor returned from load")
	case 1:
		// good, we have one index descriptor
	default:
		return nil, fmt.Errorf("more than one index descriptor returned from load")
	}

	// when we build an arch, we might have the descs for the actual arch-specific manifest, or possibly
	// an index that wraps it. So let's unwrap it and return the actual image descs and not the index.
	// this is because later we will build an index from all of these.
	r, err := c.GetContent(indexDesc[0].Digest)
	if err != nil {
		return nil, fmt.Errorf("could not get content for index descriptor: %v", err)
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	var im registry.IndexManifest
	if err := dec.Decode(&im); err != nil {
		return nil, fmt.Errorf("could not decode index descriptor: %v", err)
	}
	return im.Manifests, nil
}

type buildCtx struct {
	sources []pkgSource
	err     error
	r       io.ReadCloser
}

// Reader gets an io.Reader by iterating over the sources, tarring up the content after rewriting the paths.
// It assumes that sources is sane, ie is well formed and the first part is an absolute path
// and that it exists.
func (c *buildCtx) Reader() io.ReadCloser {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)

	go func() {
		defer func() {
			tw.Close()
			w.Close()
		}()
		for _, s := range c.sources {
			log.Debugf("Adding to build context: %s -> %s", s.src, s.dst)

			f := func(p string, i os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("ctx: Walk error on %s: %v", p, err)
				}

				var link string
				if i.Mode()&os.ModeSymlink != 0 {
					var err error
					link, err = os.Readlink(p)
					if err != nil {
						return fmt.Errorf("ctx: Failed to read symlink %s: %v", p, err)
					}
				}

				h, err := tar.FileInfoHeader(i, link)
				if err != nil {
					return fmt.Errorf("ctx: Converting FileInfo for %s: %v", p, err)
				}
				rel, err := filepath.Rel(s.src, p)
				if err != nil {
					return err
				}
				h.Name = filepath.ToSlash(filepath.Join(s.dst, rel))
				if err := tw.WriteHeader(h); err != nil {
					return fmt.Errorf("ctx: Writing header for %s: %v", p, err)
				}

				if !i.Mode().IsRegular() {
					return nil
				}

				f, err := os.Open(p)
				if err != nil {
					return fmt.Errorf("ctx: Open %s: %v", p, err)
				}
				defer f.Close()

				_, err = io.Copy(tw, f)
				if err != nil {
					return fmt.Errorf("ctx: Writing %s: %v", p, err)
				}
				return nil
			}

			if err := filepath.Walk(s.src, f); err != nil {
				c.err = err
				return
			}
		}
	}()
	c.r = r
	return c
}

// Read wraps the usual read, but allows us to include an error
func (c *buildCtx) Read(data []byte) (n int, err error) {
	if c.err != nil {
		return 0, err
	}
	return c.r.Read(data)
}

// Close wraps the usual close
func (c *buildCtx) Close() error {
	return c.r.Close()
}

// getBuilderForPlatform given an arch, find the context for the desired builder.
// If it does not exist, return "".
func getBuilderForPlatform(arch string, builders map[string]string) string {
	return builders[fmt.Sprintf("linux/%s", arch)]
}
