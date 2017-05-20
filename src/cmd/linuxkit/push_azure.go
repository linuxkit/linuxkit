package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Process the run arguments and execute run
func pushAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run azure [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("VHD image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	accountName := flags.String("accountName", "", "Azure Storage Account")
	accountKey := flags.String("accountKey", "", "Azure Storage Account Key")

	containerName := flags.String("containerName", "default-container", "Storage container name")
	blobName := flags.String("blobName", "default-linuxkit-blob", "Name of the blob to upload image")
	imageName := flags.String("imageName", "disk.vhd", "Name of the image to be uploaded")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	uploadFile(*accountName, *accountKey, *containerName, *blobName, *imageName)
}
