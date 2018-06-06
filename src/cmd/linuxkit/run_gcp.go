package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
		fmt.Printf("USAGE: %s run gcp [options] [image]\n\n", invoked)
		fmt.Printf("'image' specifies either the name of an already uploaded\n")
		fmt.Printf("GCP image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	name := flags.String("name", "", "Machine name")
	zoneFlag := flags.String("zone", defaultZone, "GCP Zone")
	machineFlag := flags.String("machine", defaultMachine, "GCP Machine Type")
	keysFlag := flags.String("keys", "", "Path to Service Account JSON key file")
	projectFlag := flags.String("project", "", "GCP Project Name")
	var disks Disks
	flags.Var(&disks, "disk", "Disk config, may be repeated. [file=]diskName[,size=1G]")

	skipCleanup := flags.Bool("skip-cleanup", false, "Don't remove images or VMs")
	nestedVirt := flags.Bool("nested-virt", false, "Enabled nested virtualization")

	data := flags.String("data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	dataPath := flags.String("data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	if *data != "" && *dataPath != "" {
		log.Fatal("Cannot specify both -data and -data-file")
	}

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the name of the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	image := remArgs[0]
	if *name == "" {
		*name = image
	}

	if *dataPath != "" {
		dataB, err := ioutil.ReadFile(*dataPath)
		if err != nil {
			log.Fatalf("Unable to read metadata file: %v", err)
		}
		*data = string(dataB)
	}

	zone := getStringValue(zoneVar, *zoneFlag, defaultZone)
	machine := getStringValue(machineVar, *machineFlag, defaultMachine)
	keys := getStringValue(keysVar, *keysFlag, "")
	project := getStringValue(projectVar, *projectFlag, "")

	client, err := NewGCPClient(keys, project)
	if err != nil {
		log.Fatalf("Unable to connect to GCP")
	}

	if err = client.CreateInstance(*name, image, zone, machine, disks, data, *nestedVirt, true); err != nil {
		log.Fatal(err)
	}

	if err = client.ConnectToInstanceSerialPort(*name, zone); err != nil {
		log.Fatal(err)
	}

	if !*skipCleanup {
		if err = client.DeleteInstance(*name, zone, true); err != nil {
			log.Fatal(err)
		}
	}
}
