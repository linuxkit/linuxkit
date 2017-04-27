package main

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

func pushUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s push [backend] [options] [prefix]\n\n", invoked)

	fmt.Printf("'backend' specifies the push backend.\n")
	fmt.Printf("Supported backends are\n")
	fmt.Printf("  gcp\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the backend specific options.\n")
	fmt.Printf("See '%s push [backend] --help' for details.\n\n", invoked)
	fmt.Printf("'prefix' specifies the path to the VM image.\n")
}

func push(args []string) {
	if len(args) < 1 {
		runUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "help", "-h", "-help", "--help":
		pushUsage()
		os.Exit(0)
	case "gcp":
		pushGcp(args[1:])
	default:
		log.Errorf("No 'push' backend specified.")
	}
}
