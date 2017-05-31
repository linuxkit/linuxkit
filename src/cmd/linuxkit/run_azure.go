package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// This program requires that the following environment vars are set:

// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret

func runAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run azure [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("Azure VM VHD or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	resourceGroupName := flags.String("resourceGroupName", "", "Name of resource group to be used for VM")
	location := flags.String("location", "westus", "Location of the VM")
	accountName := flags.String("accountName", "linuxkitstorage", "Name of the storage account")

	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

	group := createResourceGroup(*resourceGroupName, *location)
	createStorageAccount(*accountName, *location, *group)
	createVirtualNetwork(*group, "linuxkitvnetgo9", *location)
	subnet := createSubnet(*group, "linuxkitvnetgo9", "subnet")
	publicIPAddress := createPublicIPAddress(*group, "publicip46", *location)
	createNetworkInterface(*group, "linxkitNetworkInterface46", *publicIPAddress, *subnet, *location)
}
