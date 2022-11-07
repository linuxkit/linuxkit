package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
)

func pkgShowTag(args []string) {
	flags := flag.NewFlagSet("pkg show-tag", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg show-tag [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}
	canonical := flags.Bool("canonical", false, "Show canonical name, e.g. docker.io/linuxkit/foo:1234, instead of the default, e.g. linuxkit/foo:1234")

	pkgs, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	for _, p := range pkgs {
		tag := p.Tag()
		if *canonical {
			tag = p.FullTag()
		}
		fmt.Println(tag)
	}
}
