package pkgsrc

import (
	"fmt"
	"os"
	"runtime"
)

type buildOpts struct {
	force bool
	push  bool
}

// BuildOpt allows callers to specify options to Build
type BuildOpt func(bo *buildOpts) error

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

// Build builds the package
func (ps PkgSrc) Build(bos ...BuildOpt) error {
	var bo buildOpts
	for _, fn := range bos {
		if err := fn(&bo); err != nil {
			return err
		}
	}

	if _, ok := os.LookupEnv("DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"); !ok && ps.trust && bo.push {
		return fmt.Errorf("Pushing with trust enabled requires $DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE to be set")
	}

	arch := runtime.GOARCH

	if !ps.archSupported(arch) {
		return fmt.Errorf("Arch %s not supported by this package", arch)
	}
	if err := ps.cleanForBuild(); err != nil {
		return err
	}

	var suffix string
	switch arch {
	case "amd64", "arm64":
		suffix = "-" + arch
	default:
		return fmt.Errorf("Unknown arch %q", arch)
	}

	release, err := gitCommitTag("HEAD")
	if err != nil {
		return err
	}

	d := newDockerRunner(ps.trust, ps.cache)

	if !bo.force {
		ok, err := d.pull(ps.Tag() + suffix)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		fmt.Println("No image pulled, continuing with build")
	}

	var args []string

	if ps.gitRepo != "" {
		args = append(args, "--label", "org.opencontainers.image.source="+ps.gitRepo)
	}
	if !ps.dirty {
		commit, err := gitCommitHash("HEAD")
		if err != nil {
			return err
		}
		args = append(args, "--label", "org.opencontainers.image.revision="+commit)
	}

	if !ps.network {
		args = append(args, "--network=none")
	}

	if err := d.build(ps.Tag()+suffix, ps.pkgPath, args...); err != nil {
		return err
	}

	if !bo.push {
		fmt.Printf("Build complete, not pushing, all done.\n")
		return nil
	}

	if ps.dirty {
		return fmt.Errorf("build complete, refusing to push dirty package")
	}

	// If !bo.force then could do a `docker pull` here, to check
	// if there is something on hub so as not to override.
	// TODO(ijc) old make based system did this. Not sure if it
	// matters given we do either pull or build above in the
	// !force case.

	if err := d.pushWithManifest(ps.Tag(), suffix); err != nil {
		return err
	}

	if release == "" {
		fmt.Printf("Build and push complete, not releasing, all done.\n")
		return nil
	}

	relTag, err := ps.ReleaseTag(release)
	if err != nil {
		return err
	}

	if err := d.tag(ps.Tag()+suffix, relTag+suffix); err != nil {
		return err
	}

	if err := d.pushWithManifest(relTag, suffix); err != nil {
		return err
	}

	fmt.Printf("Build, push and release complete, all done.\n")

	return nil
}
