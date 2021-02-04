package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
)

func pkgBuild(args []string) {
	flags := flag.NewFlagSet("pkg build", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg build [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild")
	docker := flags.Bool("docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	buildCacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for storing built image, incompatible with --docker")

	p, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Building %q\n", p.Tag())

	opts := []pkglib.BuildOpt{pkglib.WithBuildImage()}
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	opts = append(opts, pkglib.WithBuildCacheDir(*buildCacheDir))
	if *docker {
		opts = append(opts, pkglib.WithBuildTargetDockerCache())
	}
	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
