package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Process the push arguments and execute push
func pushVCenter(args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var newVM vmConfig

	flags := flag.NewFlagSet("vCenter", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push vcenter [options] path \n\n", invoked)
		fmt.Printf("'path' specifies the full path of an ISO image. It will be pushed to a vCenter cluster.\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	newVM.vCenterURL = flags.String("url", os.Getenv("VCURL"), "URL of VMware vCenter in the format of https://username:password@VCaddress/sdk")
	newVM.dsName = flags.String("datastore", os.Getenv("VCDATASTORE"), "The name of the DataStore to host the image")
	newVM.networkName = flags.String("network", os.Getenv("VCNETWORK"), "The network label the VM will use")
	newVM.vSphereHost = flags.String("hostname", os.Getenv("VCHOST"), "The server that will host the image")
	newVM.path = flags.String("path", "", "Path to a specific image")

	newVM.vmFolder = flags.String("folder", "", "A folder on the datastore to push the image too")

	if err := flags.Parse(args); err != nil {
		log.Fatalln("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	*newVM.path = remArgs[0]

	// Ensure an iso has been passed to the vCenter push Command
	if !strings.HasSuffix(*newVM.path, ".iso") {
		log.Fatalln("Please specify an '.iso' file")
	}

	// Test any passed in files before uploading image
	checkFile(*newVM.path)

	// Connect to VMware vCenter and return the values needed to upload image
	c, dss, _, _, _, _ := vCenterConnect(ctx, newVM)

	// Create a folder from the uploaded image name if needed
	if *newVM.vmFolder == "" {
		*newVM.vmFolder = strings.TrimSuffix(path.Base(*newVM.path), ".iso")
	}

	// The CreateFolder method isn't necessary as the *newVM.vmname will be created automatically
	uploadFile(c, newVM, dss)
}
