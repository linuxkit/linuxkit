package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkgsrc"
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

	ps, err := pkgsrc.NewFromCLI(flags, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var opts []pkgsrc.BuildOpt
	opts = append(opts, pkgsrc.WithBuildPush())
	if *force {
		opts = append(opts, pkgsrc.WithBuildForce())
	}

	if err := ps.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
