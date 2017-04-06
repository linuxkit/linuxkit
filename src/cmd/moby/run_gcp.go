package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Process the run arguments and execute run
func runGcp(args []string) {
	gceCmd := flag.NewFlagSet("gce", flag.ExitOnError)
	gceCmd.Usage = func() {
		fmt.Printf("USAGE: %s run gce [options] [name]\n\n", os.Args[0])
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("GCE image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		gceCmd.PrintDefaults()
	}
	zone := gceCmd.String("zone", "europe-west1-d", "GCE Zone")
	machine := gceCmd.String("machine", "g1-small", "GCE Machine Type")
	keys := gceCmd.String("keys", "", "Path to Service Account JSON key file")
	project := gceCmd.String("project", "", "GCE Project Name")
	bucket := gceCmd.String("bucket", "", "GS Bucket to upload to. *Required* when 'prefix' is a filename")
	public := gceCmd.Bool("public", false, "Select if file on GS should be public. *Optional* when 'prefix' is a filename")
	family := gceCmd.String("family", "", "GCE Image Family. A group of images where the family name points to the most recent image. *Optional* when 'prefix' is a filename")

	gceCmd.Parse(args)
	remArgs := gceCmd.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the prefix to the image to boot\n")
		gceCmd.Usage()
		os.Exit(1)
	}
	prefix := remArgs[0]

	client, err := NewGCEClient(*keys, *project)
	if err != nil {
		log.Fatalf("Unable to connect to GCE")
	}

	suffix := ".img.tar.gz"
	if strings.HasSuffix(prefix, suffix) {
		filename := prefix
		prefix = prefix[:len(prefix)-len(suffix)]
		if *bucket == "" {
			log.Fatalf("No bucket specified. Please provide one using the -bucket flag")
		}
		err = client.UploadFile(filename, *bucket, *public)
		if err != nil {
			log.Fatalf("Error copying to Google Storage: %v", err)
		}
		err = client.CreateImage(prefix, "https://storage.googleapis.com/"+*bucket+"/"+prefix+".img.tar.gz", *family, true)
		if err != nil {
			log.Fatalf("Error creating Google Compute Image: %v", err)
		}
	}

	if err = client.CreateInstance(prefix, *zone, *machine, true); err != nil {
		log.Fatal(err)
	}

	if err = client.ConnectToInstanceSerialPort(prefix, *zone); err != nil {
		log.Fatal(err)
	}

	if err = client.DeleteInstance(prefix, *zone, true); err != nil {
		log.Fatal(err)
	}
}
