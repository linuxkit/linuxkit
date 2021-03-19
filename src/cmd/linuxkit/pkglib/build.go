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

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	minimumDockerVersion = "19.03"
)

type buildOpts struct {
	skipBuild    bool
	force        bool
	push         bool
	release      string
	manifest     bool
	image        bool
	targetDocker bool
	cache        string
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

// WithBuildImage builds the image
func WithBuildImage() BuildOpt {
	return func(bo *buildOpts) error {
		bo.image = true
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
		bo.cache = dir
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

	arch := runtime.GOARCH

	if !p.archSupported(arch) {
		fmt.Printf("Arch %s not supported by this package, skipping build.\n", arch)
		return nil
	}
	if err := p.cleanForBuild(); err != nil {
		return err
	}

	var (
		desc   *v1.Descriptor
		suffix string
	)
	switch arch {
	case "amd64", "arm64", "s390x":
		suffix = "-" + arch
	default:
		return fmt.Errorf("Unknown arch %q", arch)
	}

	// did we have the build cache dir provided? Yes, there is a default, but that is at the CLI level,
	// and expected to be provided at this function level
	if bo.cache == "" && !bo.targetDocker {
		return errors.New("must provide linuxkit build cache directory when not targeting docker")
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

	d := newDockerRunner(p.cache)

	if err := d.buildkitCheck(); err != nil {
		return fmt.Errorf("buildkit not supported, check docker version: %v", err)
	}

	if !bo.force {
		if bo.targetDocker {
			ok, err := d.pull(p.Tag())
			// any error returns
			if err != nil {
				return err
			}
			// if we already have it, do not bother building any more
			if ok {
				return nil
			}
		} else {
			ref, err := reference.Parse(p.Tag())
			if err != nil {
				return fmt.Errorf("could not resolve references for image %s: %v", p.Tag(), err)
			}
			if _, err := cache.ImageWrite(bo.cache, &ref, "", arch); err == nil {
				fmt.Printf("image already found %s", ref)
				return nil
			}
		}
		fmt.Println("No image pulled, continuing with build")
	}

	if bo.image && !bo.skipBuild {
		var args []string

		if err := p.dockerDepends.Do(d); err != nil {
			return err
		}

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

		d.ctx = &buildCtx{sources: p.sources}

		// set the target
		var (
			buildxOutput string
			stdout       io.WriteCloser
			tag          = p.Tag()
			tagArch      = tag + suffix
			eg           errgroup.Group
			stdoutCloser = func() {
				if stdout != nil {
					stdout.Close()
				}
			}
		)
		ref, err := reference.Parse(tag)
		if err != nil {
			return fmt.Errorf("could not resolve references for image %s: %v", tagArch, err)
		}

		if bo.targetDocker {
			buildxOutput = "type=docker"
			stdout = nil
			// there is no gofunc processing for simple output to docker
		} else {
			// we are writing to local, so we need to catch the tar output stream and place the right files in the right place
			buildxOutput = "type=oci"
			piper, pipew := io.Pipe()
			stdout = pipew

			eg.Go(func() error {
				source, err := cache.ImageWriteTar(bo.cache, &ref, arch, piper)
				// send the error down the channel
				if err != nil {
					fmt.Printf("cache.ImageWriteTar goroutine ended with error: %v\n", err)
				}
				desc = source.Descriptor()
				piper.Close()
				return err
			})
		}
		args = append(args, fmt.Sprintf("--output=%s", buildxOutput))

		if err := d.build(tagArch, p.path, stdout, args...); err != nil {
			stdoutCloser()
			return err
		}
		stdoutCloser()

		// wait for the processor to finish
		if err := eg.Wait(); err != nil {
			return err
		}

		// create the arch-less image
		switch {
		case bo.targetDocker:
			// if in docker, use a tag
			if err := d.tag(tagArch, tag); err != nil {
				return err
			}
		case desc == nil:
			return errors.New("no valid descriptor returned for image")
		default:
			// if in the proper linuxkit cache, create a multi-arch index
			if _, err := cache.IndexWrite(bo.cache, &ref, *desc); err != nil {
				return err
			}
		}

		if !bo.push {
			fmt.Printf("Build complete, not pushing, all done.\n")
			return nil
		}
	}

	if p.dirty {
		return fmt.Errorf("build complete, refusing to push dirty package")
	}

	// If !bo.force then could do a `docker pull` here, to check
	// if there is something on hub so as not to override.
	// TODO(ijc) old make based system did this. Not sure if it
	// matters given we do either pull or build above in the
	// !force case.

	if bo.targetDocker {
		if err := d.pushWithManifest(p.Tag(), suffix, bo.image, bo.manifest); err != nil {
			return err
		}
	} else {
		if err := cache.PushWithManifest(bo.cache, p.Tag(), suffix, bo.image, bo.manifest); err != nil {
			return err
		}
	}

	if bo.release == "" {
		fmt.Printf("Build and push complete, not releasing, all done.\n")
		return nil
	}

	relTag, err := p.ReleaseTag(bo.release)
	if err != nil {
		return err
	}

	if bo.targetDocker {
		if err := d.tag(p.Tag()+suffix, relTag+suffix); err != nil {
			return err
		}

		if err := d.pushWithManifest(relTag, suffix, bo.image, bo.manifest); err != nil {
			return err
		}
	} else {
		// must make sure descriptor is available
		if desc == nil {
			desc, err = cache.FindDescriptor(bo.cache, p.Tag()+suffix)
			if err != nil {
				return err
			}
		}
		ref, err := reference.Parse(relTag + suffix)
		if err != nil {
			return err
		}
		if _, err := cache.DescriptorWrite(bo.cache, &ref, *desc); err != nil {
			return err
		}
		if err := cache.PushWithManifest(bo.cache, relTag, suffix, bo.image, bo.manifest); err != nil {
			return err
		}
	}

	fmt.Printf("Build, push and release of %q complete, all done.\n", bo.release)

	return nil
}

type buildCtx struct {
	sources []pkgSource
}

// Copy iterates over the sources, tars up the content after rewriting the paths.
// It assumes that sources is sane, ie is well formed and the first part is an absolute path
// and that it exists. NewFromCLI() ensures that.
func (c *buildCtx) Copy(w io.WriteCloser) error {
	tw := tar.NewWriter(w)
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
			return err
		}
	}

	return nil
}
