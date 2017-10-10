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
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg build [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild")
	release := flags.String("release", "", "Release the given version")

	p, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var opts []pkglib.BuildOpt
	opts = append(opts, pkglib.WithBuildPush())
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	if *release != "" {
		opts = append(opts, pkglib.WithRelease(*release))
	}
	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
