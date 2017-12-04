package pkglib

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
)

type buildOpts struct {
	skipBuild bool
	force     bool
	push      bool
	release   string
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

// WithRelease releases as the given version after push
func WithRelease(r string) BuildOpt {
	return func(bo *buildOpts) error {
		bo.release = r
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

	if _, ok := os.LookupEnv("DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"); !ok && p.trust && bo.push {
		return fmt.Errorf("Pushing with trust enabled requires $DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE to be set")
	}

	arch := runtime.GOARCH

	if !p.archSupported(arch) {
		fmt.Printf("Arch %s not supported by this package, skipping build.\n", arch)
		return nil
	}
	if err := p.cleanForBuild(); err != nil {
		return err
	}

	var suffix string
	switch arch {
	case "amd64", "arm64":
		suffix = "-" + arch
	default:
		return fmt.Errorf("Unknown arch %q", arch)
	}

	if p.git != nil && bo.release == "" {
		r, err := p.git.commitTag("HEAD")
		if err != nil {
			return err
		}
		bo.release = r
	}

	if bo.release != "" && !bo.push {
		return fmt.Errorf("Cannot release %q if not pushing", bo.release)
	}

	d := newDockerRunner(p.trust, p.cache)

	if !bo.force {
		ok, err := d.pull(p.Tag())
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		fmt.Println("No image pulled, continuing with build")
	}

	if !bo.skipBuild {
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

		if err := d.build(p.Tag()+suffix, p.path, args...); err != nil {
			return err
		}

		if !bo.push {
			if err := d.tag(p.Tag()+suffix, p.Tag()); err != nil {
				return err
			}

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

	if err := d.pushWithManifest(p.Tag(), suffix); err != nil {
		return err
	}

	if bo.release == "" {
		fmt.Printf("Build and push complete, not releasing, all done.\n")
		return nil
	}

	relTag, err := p.ReleaseTag(bo.release)
	if err != nil {
		return err
	}

	if err := d.tag(p.Tag()+suffix, relTag+suffix); err != nil {
		return err
	}

	if err := d.pushWithManifest(relTag, suffix); err != nil {
		return err
	}

	fmt.Printf("Build, push and release of %q complete, all done.\n", bo.release)

	return nil
}
