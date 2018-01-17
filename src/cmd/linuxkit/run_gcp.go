package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	defaultZone    = "europe-west1-d"
	defaultMachine = "g1-small"
	// Environment variables. Some are non-standard
	zoneVar    = "CLOUDSDK_COMPUTE_ZONE"
	machineVar = "CLOUDSDK_COMPUTE_MACHINE" // non-standard
	keysVar    = "CLOUDSDK_COMPUTE_KEYS"    // non-standard
	projectVar = "CLOUDSDK_CORE_PROJECT"
	bucketVar  = "CLOUDSDK_IMAGE_BUCKET" // non-standard
	familyVar  = "CLOUDSDK_IMAGE_FAMILY" // non-standard
	publicVar  = "CLOUDSDK_IMAGE_PUBLIC" // non-standard
	nameVar    = "CLOUDSDK_IMAGE_NAME"   // non-standard
)

// Process the run arguments and execute run
func runGcp(args []string) {
	flags := flag.NewFlagSet("gcp", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run gcp [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("GCP image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	zoneFlag := flags.String("zone", defaultZone, "GCP Zone")
	machineFlag := flags.String("machine", defaultMachine, "GCP Machine Type")
	keysFlag := flags.String("keys", "", "Path to Service Account JSON key file")
	projectFlag := flags.String("project", "", "GCP Project Name")
	var disks Disks
	flags.Var(&disks, "disk", "Disk config, may be repeated. [file=]diskName[,size=1G]")

	skipCleanup := flags.Bool("skip-cleanup", false, "Don't remove images or VMs")
	nestedVirt := flags.Bool("nested-virt", false, "Enabled nested virtualization")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the name of the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	name := remArgs[0]

	zone := getStringValue(zoneVar, *zoneFlag, defaultZone)
	machine := getStringValue(machineVar, *machineFlag, defaultMachine)
	keys := getStringValue(keysVar, *keysFlag, "")
	project := getStringValue(projectVar, *projectFlag, "")

	client, err := NewGCPClient(keys, project)
	if err != nil {
		log.Fatalf("Unable to connect to GCP")
	}

	if err = client.CreateInstance(name, name, zone, machine, disks, *nestedVirt, true); err != nil {
		log.Fatal(err)
	}

	if err = client.ConnectToInstanceSerialPort(name, zone); err != nil {
		log.Fatal(err)
	}

	if !*skipCleanup {
		if err = client.DeleteInstance(name, zone, true); err != nil {
			log.Fatal(err)
		}
	}
}
