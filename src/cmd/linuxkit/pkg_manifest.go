package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
)

func pkgManifest(args []string) {
	pkgIndex(args)
}
func pkgIndex(args []string) {
	flags := flag.NewFlagSet("pkg manifest", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		name := "manifest"
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg %s [options] path\n\n", name, invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Updates the manifest in the registry for the given path based on all known platforms. If none found, no manifest created.\n")
		flags.PrintDefaults()
	}
	release := flags.String("release", "", "Release the given version")

	pkgs, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var opts []pkglib.BuildOpt
	if *release != "" {
		opts = append(opts, pkglib.WithRelease(*release))
	}

	for _, p := range pkgs {
		msg := fmt.Sprintf("Updating index for %q", p.Tag())
		action := "building and pushing"

		fmt.Println(msg)

		if err := p.Index(opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error %s %q: %v\n", action, p.Tag(), err)
			os.Exit(1)
		}
	}
}
