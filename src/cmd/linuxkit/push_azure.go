package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// In order to run this, you need to set the following enrivonment variables:

// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID

// Process the run arguments and execute run
func pushAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run gcp [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("GCP image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	accountName := flags.String("accountName", "", "Azure Storage Account")
	accountKey := flags.String("accountKey", "", "Azure Storage Account Key")

	containerName := flags.String("containerName", "default-container", "Storage container name")

	client, err := storage.NewBasicClient(*accountName, *accountKey)
	if err != nil {
		log.Fatalf("Unable to create storage client")
	}

	blobClient := client.GetBlobService()
	options := storage.CreateContainerOptions{}
	container := blobClient.GetContainerReference(*containerName)

	_, err = container.CreateIfNotExists(&options)
	if err != nil {
		log.Fatalf("Unable to create storage container")
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
