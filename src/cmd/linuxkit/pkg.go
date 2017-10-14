package main

import (
	"fmt"
	"os"
	"os/exec"
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

func setupContentTrustPassphrase() {
	// If it is already set there is nothing to do.
	if _, ok := os.LookupEnv("DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"); ok {
		return
	}
	// If it is not set but it is needed this is checked at time
	// of use, not all commands need it.
	if Config.Pkg.ContentTrustCommand == "" {
		return
	}

	// Run the command and set the output as the passphrase
	cmd := exec.Command("/bin/sh", "-c", Config.Pkg.ContentTrustCommand)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	v, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to run ContentTrustCommand: %s\n", err)
		os.Exit(1)
	}
	os.Setenv("DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE", string(v))
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
