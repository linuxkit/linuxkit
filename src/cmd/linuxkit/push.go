package main

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func pushUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s push [backend] [options] [prefix]\n\n", invoked)
	fmt.Printf("'backend' specifies the push backend.\n")
	fmt.Printf("Supported backends are\n")
	// Please keep these in alphabetical order
	fmt.Printf("  aws\n")
	fmt.Printf("  azure\n")
	fmt.Printf("  gcp\n")
	fmt.Printf("  openstack\n")
	fmt.Printf("  packet\n")
	fmt.Printf("  scaleway\n")
	fmt.Printf("  vcenter\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the backend specific options.\n")
	fmt.Printf("See '%s push [backend] --help' for details.\n\n", invoked)
	fmt.Printf("'prefix' specifies the path to the VM image.\n")
}

func push(args []string) {
	if len(args) < 1 {
		pushUsage()
		os.Exit(1)
	}

	switch args[0] {
	// Please keep cases in alphabetical order
	case "aws":
		pushAWS(args[1:])
	case "azure":
		pushAzure(args[1:])
	case "gcp":
		pushGcp(args[1:])
	case "openstack":
		pushOpenstack(args[1:])
	case "packet":
		pushPacket(args[1:])
	case "scaleway":
		pushScaleway(args[1:])
	case "vcenter":
		pushVCenter(args[1:])
	case "help", "-h", "-help", "--help":
		pushUsage()
		os.Exit(0)
	default:
		log.Errorf("No 'push' backend specified.")
	}
}
