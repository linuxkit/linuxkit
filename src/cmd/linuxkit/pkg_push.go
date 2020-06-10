package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
)

func pkgPush(args []string) {
	flags := flag.NewFlagSet("pkg push", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg push [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild")
	release := flags.String("release", "", "Release the given version")
	nobuild := flags.Bool("nobuild", false, "Skip the build")
	manifest := flags.Bool("manifest", true, "Create and push multi-arch manifest")
	image := flags.Bool("image", true, "Build and push image for the current platform")
	sign := flags.Bool("sign", true, "sign the manifest, if a manifest is created; ignored if --manifest=false")

	p, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if p.TrustEnabled() {
		setupContentTrustPassphrase()
	}

	var opts []pkglib.BuildOpt
	opts = append(opts, pkglib.WithBuildPush())
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	if *nobuild {
		opts = append(opts, pkglib.WithBuildSkip())
	}
	if *release != "" {
		opts = append(opts, pkglib.WithRelease(*release))
	}
	if *manifest {
		opts = append(opts, pkglib.WithBuildManifest())
	}
	if *image {
		opts = append(opts, pkglib.WithBuildImage())
	}
	// only sign manifests; ignore for image only
	if *sign && *manifest {
		opts = append(opts, pkglib.WithBuildSign())
	}

	if *nobuild {
		fmt.Printf("Pushing %q without building\n", p.Tag())
	} else {
		fmt.Printf("Building and pushing %q\n", p.Tag())
	}

	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
