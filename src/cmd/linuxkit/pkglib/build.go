package pkglib

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/containerd/containerd/reference"
	registry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	minimumDockerVersion = "19.03"
)

type buildOpts struct {
	skipBuild     bool
	force         bool
	push          bool
	release       string
	manifest      bool
	image         bool
	targetDocker  bool
	cacheDir      string
	cacheProvider lktspec.CacheProvider
	platforms     []imagespec.Platform
	builders      map[string]string
	runner        dockerRunner
	writer        io.Writer
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

// Build builds the package
func (p Pkg) Build(bos ...BuildOpt) error {
	var bo buildOpts
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

	// if targeting docker, be sure local arch is a build target
	if bo.targetDocker {
		var found bool
		for _, platform := range bo.platforms {
			if platform.Architecture == arch {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("must build for local platform 'linux/%s' when targeting docker", arch)
		}
	}

	if p.git != nil && bo.push && bo.release == "" {
		r, err := p.git.commitTag("HEAD")
		if err != nil {
			return err
		}
		bo.release = r
	}

	if bo.release != "" && !bo.push {
		return fmt.Errorf("Cannot release %q if not pushing", bo.release)
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

	if err := d.buildkitCheck(); err != nil {
		return fmt.Errorf("buildkit not supported, check docker version: %v", err)
	}

	skipBuild := bo.skipBuild
	if !bo.force {
		fmt.Fprintf(writer, "checking for %s in local cache, fallback to remote registry...\n", ref)
		if _, err := c.ImagePull(&ref, "", arch, false); err == nil {
			fmt.Fprintf(writer, "%s found or pulled\n", ref)
			skipBuild = true
		} else {
			fmt.Fprintf(writer, "%s not found\n", ref)
		}
	}

	if !skipBuild {
		fmt.Fprintf(writer, "building %s\n", ref)
		var (
			args  []string
			descs []registry.Descriptor
		)

		if p.git != nil && p.gitRepo != "" {
			args = append(args, "--label", "org.opencontainers.image.source="+p.gitRepo)
		}
		if p.git != nil && !p.dirty {
			commit, err := p.git.commitHash("HEAD")
			if err != nil {
				return err
			}
			args = append(args, "--label", "org.opencontainers.image.revision="+commit)
		}

		if !p.network {
			args = append(args, "--network=none")
		}

		if p.config != nil {
			b, err := json.Marshal(*p.config)
			if err != nil {
				return err
			}
			args = append(args, "--label=org.mobyproject.config="+string(b))
		}

		args = append(args, "--label=org.mobyproject.linuxkit.version="+version.Version)
		args = append(args, "--label=org.mobyproject.linuxkit.revision="+version.GitCommit)

		// build for each arch and save in the linuxkit cache
		for _, platform := range bo.platforms {
			desc, err := p.buildArch(d, c, platform.Architecture, args, writer, bo)
			if err != nil {
				return fmt.Errorf("error building for arch %s: %v", platform.Architecture, err)
			}
			if desc == nil {
				return fmt.Errorf("no valid descriptor returned for image for arch %s", platform.Architecture)
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
	desc, err := c.FindDescriptor(p.FullTag())
	if err != nil {
		return err
	}

	// if requested docker, load the image up
	if bo.targetDocker {
		cacheSource := c.NewSource(&ref, arch, desc)
		reader, err := cacheSource.V1TarReader()
		if err != nil {
			return fmt.Errorf("unable to get reader from cache: %v", err)
		}
		if err := d.load(reader); err != nil {
			return err
		}
	}

	if !bo.push {
		fmt.Fprintf(writer, "Build complete, not pushing, all done.\n")
		return nil
	}

	if p.dirty {
		return fmt.Errorf("build complete, refusing to push dirty package")
	}

	// push the manifest
	if err := c.Push(p.FullTag()); err != nil {
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
	if err := c.Push(fullRelTag); err != nil {
		return err
	}

	// tag in docker, if requested
	if bo.targetDocker {
		if err := d.tag(p.FullTag(), fullRelTag); err != nil {
			return err
		}
	}

	fmt.Fprintf(writer, "Build, push and release of %q complete, all done.\n", bo.release)

	return nil
}

// buildArch builds the package for a single arch
func (p Pkg) buildArch(d dockerRunner, c lktspec.CacheProvider, arch string, args []string, writer io.Writer, bo buildOpts) (*registry.Descriptor, error) {
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
			desc, err := c.FindDescriptor(ref.String())
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
		buildxOutput string
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
	buildxOutput = "type=oci"
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
	args = append(args, fmt.Sprintf("--output=%s", buildxOutput))

	buildCtx := &buildCtx{sources: p.sources}
	platform := fmt.Sprintf("linux/%s", arch)
	archArgs := append(args, "--platform")
	archArgs = append(archArgs, platform)
	if err := d.build(tagArch, p.path, builderName, platform, buildCtx.Reader(), stdout, archArgs...); err != nil {
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
// and that it exists. NewFromCLI() ensures that.
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
