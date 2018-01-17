package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func pushGcp(args []string) {
	flags := flag.NewFlagSet("gcp", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push gcp [options] path\n\n", invoked)
		fmt.Printf("'path' is the full path to a GCP image. It will be uploaded to GCS and GCP VM image will be created from it.\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	keysFlag := flags.String("keys", "", "Path to Service Account JSON key file")
	projectFlag := flags.String("project", "", "GCP Project Name")
	bucketFlag := flags.String("bucket", "", "GCS Bucket to upload to. *Required*")
	publicFlag := flags.Bool("public", false, "Select if file on GCS should be public. *Optional*")
	familyFlag := flags.String("family", "", "GCP Image Family. A group of images where the family name points to the most recent image. *Optional*")
	nameFlag := flags.String("img-name", "", "Overrides the name used to identify the file in Google Storage and the VM image. Defaults to the base of 'path' with the '.img.tar.gz' suffix removed")
	nestedVirt := flags.Bool("nested-virt", false, "Enabled nested virtualization for the image")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]

	keys := getStringValue(keysVar, *keysFlag, "")
	project := getStringValue(projectVar, *projectFlag, "")
	bucket := getStringValue(bucketVar, *bucketFlag, "")
	public := getBoolValue(publicVar, *publicFlag)
	family := getStringValue(familyVar, *familyFlag, "")
	name := getStringValue(nameVar, *nameFlag, "")

	const suffix = ".img.tar.gz"
	if name == "" {
		name = strings.TrimSuffix(path, suffix)
		name = filepath.Base(name)
	}

	if bucket == "" {
		log.Fatalf("Please specify the bucket to use")
	}

	client, err := NewGCPClient(keys, project)
	if err != nil {
		log.Fatalf("Unable to connect to GCP: %v", err)
	}

	err = client.UploadFile(path, name+suffix, bucket, public)
	if err != nil {
		log.Fatalf("Error copying to Google Storage: %v", err)
	}
	err = client.CreateImage(name, "https://storage.googleapis.com/"+bucket+"/"+name+suffix, family, *nestedVirt, true)
	if err != nil {
		log.Fatalf("Error creating Google Compute Image: %v", err)
	}
}
