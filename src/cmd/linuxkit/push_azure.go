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
		fmt.Printf("USAGE: %s push azure [options] path\n\n", invoked)
		fmt.Printf("Push a disk image to Azure\n")
		fmt.Printf("'path' specifies the path to a VHD. It will be uploaded to an Azure Storage Account.\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	resourceGroup := flags.String("resource-group", "", "Name of resource group to be used for VM")
	accountName := flags.String("storage-account", "", "Name of the storage account")

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

	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

	initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

	uploadVMImage(*resourceGroup, *accountName, path)
}
