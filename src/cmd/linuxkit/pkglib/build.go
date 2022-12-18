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
	skipBuild      bool
	force          bool
	pull           bool
	ignoreCache    bool
	push           bool
	release        string
	manifest       bool
	targetDocker   bool
	cacheDir       string
	cacheProvider  lktspec.CacheProvider
	platforms      []imagespec.Platform
	builders       map[string]string
	runner         dockerRunner
	writer         io.Writer
	builderImage   string
	builderRestart bool
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
		if p.git != nil && !p.dirty {
			commit, err := p.git.commitHash("HEAD")
			if err != nil {
				return err
			}
			imageBuildOpts.Labels["org.opencontainers.image.revision"] = commit
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

		if p.buildArgs != nil {
			for _, buildArg := range *p.buildArgs {
				parts := strings.SplitN(buildArg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", buildArg)
				}
				imageBuildOpts.BuildArgs[parts[0]] = &parts[1]
			}
		}

		// build for each arch and save in the linuxkit cache
		for _, platform := range platformsToBuild {
			desc, err := p.buildArch(ctx, d, c, bo.builderImage, platform.Architecture, bo.builderRestart, writer, bo, imageBuildOpts)
			if err != nil {
				return fmt.Errorf("error building for arch %s: %v", platform.Architecture, err)
			}
			if desc == nil {
				return fmt.Errorf("no valid descriptor returned for image for arch %s", platform.Architecture)
			}
			if desc.Platform == nil {
				return fmt.Errorf("descriptor for platform %v has no information on the platform: %#v", platform, desc)
			}
			descs = append(descs, *desc)
		}

		// after build is done:
		// - create multi-arch manifest
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
	if len(platformsToBuild) == 0 && !imageInLocalCache {
		fmt.Fprintf(writer, "No new platforms to push, skipping.\n")
		return nil
	}

	if p.dirty {
		return fmt.Errorf("build complete, refusing to push dirty package")
	}

	// push the manifest
	if err := c.Push(p.FullTag(), bo.manifest); err != nil {
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
	if err := c.Push(fullRelTag, bo.manifest); err != nil {
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

// buildArch builds the package for a single arch
func (p Pkg) buildArch(ctx context.Context, d dockerRunner, c lktspec.CacheProvider, builderImage, arch string, restart bool, writer io.Writer, bo buildOpts, imageBuildOpts types.ImageBuildOptions) (*registry.Descriptor, error) {
	var (
		desc    *registry.Descriptor
		tagArch string
		tag     = p.Tag()
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
			return desc, nil
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
	ref, err := reference.Parse(p.FullTag())
	if err != nil {
		return nil, fmt.Errorf("could not resolve references for image %s: %v", tagArch, err)
	}

	// we are writing to local, so we need to catch the tar output stream and place the right files in the right place
	piper, pipew := io.Pipe()
	stdout = pipew

	eg.Go(func() error {
		source, err := c.ImageLoad(&ref, arch, piper)
		// send the error down the channel
		if err != nil {
			fmt.Fprintf(stdout, "cache.ImageLoad goroutine ended with error: %v\n", err)
		} else {
			desc = source.Descriptor()
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
	if err := d.build(ctx, tagArch, p.path, builderName, builderImage, platform, restart, passCache, buildCtx.Reader(), stdout, imageBuildOpts); err != nil {
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

	return desc, nil
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
