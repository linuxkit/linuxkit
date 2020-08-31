package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
)

type stringSliceFlag []string

func (i *stringSliceFlag) String() string {
	return fmt.Sprintf("[%s]", strings.Join(*i, " "))
}

func (i *stringSliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var buildArgs stringSliceFlag

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
	flags.Var(&buildArgs, "build-arg", "Pass runtime build arguments")

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
	for _, b := range buildArgs {
		opts = append(opts, pkglib.WithBuildArg(b))
	}
	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
