package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func pkgUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s pkg [subcommand] [options] [prefix]\n\n", invoked)

	fmt.Printf("'subcommand' is one of:\n")
	fmt.Printf("  build\n")
	fmt.Printf("  push\n")
	fmt.Printf("  show-tag\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the command specific options.\n")
	fmt.Printf("See '%s pkg [command] --help' for details.\n\n", invoked)
}

func pkg(args []string) {
	if len(args) < 1 {
		pkgUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "build":
		pkgBuild(args[1:])
	case "push":
		pkgPush(args[1:])
	case "show-tag":
		pkgShowTag(args[1:])
	default:
		fmt.Printf("Unknown subcommand %q\n\n", args[0])
		pkgUsage()
	}
}
