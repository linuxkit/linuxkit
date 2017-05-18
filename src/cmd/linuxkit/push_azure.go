package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/radu-matei/azure-sdk-for-go/storage"
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

	client, err := storage.NewBasicClient(*accountName, *accountKey)
	if err != nil {
		log.Fatalf("Unable to create storage client")
	}

	blobClient := client.GetBlobService()
	options := storage.CreateContainerOptions{}
	container := blobClient.GetContainerReference(*containerName)

	_, err = container.CreateIfNotExists(&options)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Unable to create storage container")
	}

	blob := container.GetBlobReference(*blobName)

	file := newFile(*imageName)
	defer file.Close()

	reader := bufio.NewReader(file)
	stat, err := file.Stat()
	size := int(stat.Size())

	err = blob.CreateBlockBlobFromReader(reader, nil, size)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Unable to create block blob from reader")
	}
}

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		fmt.Printf("%s missing! Exiting...\n", varName)
		os.Exit(1)
	}

	return value
}

func newFile(fn string) *os.File {
	fp, err := os.OpenFile(fn, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return fp
}
