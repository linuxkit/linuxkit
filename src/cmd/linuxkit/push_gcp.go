package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Process the run arguments and execute run
func pushGcp(args []string) {
	gcpCmd := flag.NewFlagSet("gcp", flag.ExitOnError)
	gcpCmd.Usage = func() {
		fmt.Printf("USAGE: %s push gcp [options] [name]\n\n", os.Args[0])
		fmt.Printf("'name' specifies the full path of an image file which will be uploaded\n")
		fmt.Printf("Options:\n\n")
		gcpCmd.PrintDefaults()
	}
	keysFlag := gcpCmd.String("keys", "", "Path to Service Account JSON key file")
	projectFlag := gcpCmd.String("project", "", "GCP Project Name")
	bucketFlag := gcpCmd.String("bucket", "", "GS Bucket to upload to. *Required*")
	publicFlag := gcpCmd.Bool("public", false, "Select if file on GS should be public. *Optional*")
	familyFlag := gcpCmd.String("family", "", "GCP Image Family. A group of images where the family name points to the most recent image. *Optional*")
	nameFlag := gcpCmd.String("img-name", "", "Overrides the Name used to identify the file in Google Storage and Image. Defaults to [name]")

	if err := gcpCmd.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := gcpCmd.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the prefix to the image to push\n")
		gcpCmd.Usage()
		os.Exit(1)
	}
	prefix := remArgs[0]

	keys := getStringValue(keysVar, *keysFlag, "")
	project := getStringValue(projectVar, *projectFlag, "")
	bucket := getStringValue(bucketVar, *bucketFlag, "")
	public := getBoolValue(publicVar, *publicFlag)
	family := getStringValue(familyVar, *familyFlag, "")
	name := getStringValue(nameVar, *nameFlag, "")

	client, err := NewGCPClient(keys, project)
	if err != nil {
		log.Fatalf("Unable to connect to GCP")
	}

	suffix := ".img.tar.gz"
	src := prefix
	if strings.HasSuffix(prefix, suffix) {
		prefix = prefix[:len(prefix)-len(suffix)]
	} else {
		src = prefix + suffix
	}
	if name != "" {
		prefix = name
	}
	if bucket == "" {
		log.Fatalf("No bucket specified. Please provide one using the -bucket flag")
	}
	err = client.UploadFile(src, prefix+suffix, bucket, public)
	if err != nil {
		log.Fatalf("Error copying to Google Storage: %v", err)
	}
	err = client.CreateImage(prefix, "https://storage.googleapis.com/"+bucket+"/"+prefix+".img.tar.gz", family, true)
	if err != nil {
		log.Fatalf("Error creating Google Compute Image: %v", err)
	}
}
