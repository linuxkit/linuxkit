package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
)

func runUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s run [backend] [options] [prefix]\n\n", invoked)

	fmt.Printf("'backend' specifies the run backend.\n")
	fmt.Printf("If not specified the platform specific default will be used\n")
	fmt.Printf("Supported backends are (default platform in brackets):\n")
	fmt.Printf("  azure\n")
	fmt.Printf("  gcp\n")
	fmt.Printf("  hyperkit [macOS]\n")
	fmt.Printf("  qemu [linux]\n")
	fmt.Printf("  vcenter\n")
	fmt.Printf("  vmware\n")
	fmt.Printf("  packet\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the backend specific options.\n")
	fmt.Printf("See '%s run [backend] --help' for details.\n\n", invoked)
	fmt.Printf("'prefix' specifies the path to the VM image.\n")
	fmt.Printf("It defaults to './image'.\n")
}

func run(args []string) {
	if len(args) < 1 {
		runUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "help", "-h", "-help", "--help":
		runUsage()
		os.Exit(0)
	case "hyperkit":
		runHyperKit(args[1:])
	case "azure":
		runAzure(args[1:])
	case "vmware":
		runVMware(args[1:])
	case "gcp":
		runGcp(args[1:])
	case "qemu":
		runQemu(args[1:])
	case "packet":
		runPacket(args[1:])
	case "vcenter":
		runVcenter(args[1:])
	default:
		switch runtime.GOOS {
		case "darwin":
			runHyperKit(args)
		case "linux":
			runQemu(args)
		default:
			log.Errorf("There currently is no default 'run' backend for your platform.")
		}
	}
}
