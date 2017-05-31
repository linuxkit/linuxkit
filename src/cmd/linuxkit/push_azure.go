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
		fmt.Printf("USAGE: %s push azure [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies the path (absolute or relative) of a\n")
		fmt.Printf("VHD image be uploaded to an existing Azure Storage Account\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	resourceGroupName := flags.String("resourceGroupName", "", "Name of resource group to be used for VM")
	accountName := flags.String("accountName", "linuxkitstorage", "Name of the storage account")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	imagePath := remArgs[0]

	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

	initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

	uploadVMImage(*resourceGroupName, *accountName, imagePath)
}
