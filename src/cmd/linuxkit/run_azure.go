package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// This program requires that the following environment vars are set:

// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret

const defaultStorageAccountName = "linuxkit"

func runAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run azure [options] imagePath\n\n", invoked)
		fmt.Printf("'imagePath' specifies the path (absolute or relative) of a\n")
		fmt.Printf("VHD image be used as the OS image for the VM\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	resourceGroupName := flags.String("resourceGroupName", "", "Name of resource group to be used for VM")
	location := flags.String("location", "westus", "Location of the VM")
	accountName := flags.String("accountName", defaultStorageAccountName, "Name of the storage account")

	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

	if err := flags.Parse(args); err != nil {
		log.Fatalf("Unable to parse args: %s", err.Error())
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the image to run\n")
		flags.Usage()
		os.Exit(1)
	}
	imagePath := remArgs[0]

	rand.Seed(time.Now().UTC().UnixNano())
	virtualNetworkName := fmt.Sprintf("linuxkitvirtualnetwork%d", rand.Intn(1000))
	subnetName := fmt.Sprintf("linuxkitsubnet%d", rand.Intn(1000))
	publicIPAddressName := fmt.Sprintf("publicip%d", rand.Intn(1000))
	networkInterfaceName := fmt.Sprintf("networkinterface%d", rand.Intn(1000))
	virtualMachineName := fmt.Sprintf("linuxkitvm%d", rand.Intn(1000))

	initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

	group := createResourceGroup(*resourceGroupName, *location)
	createStorageAccount(*accountName, *location, *group)
	uploadVMImage(*group.Name, *accountName, imagePath)
	createVirtualNetwork(*group, virtualNetworkName, *location)
	subnet := createSubnet(*group, virtualNetworkName, subnetName)
	publicIPAddress := createPublicIPAddress(*group, publicIPAddressName, *location)
	networkInterface := createNetworkInterface(*group, networkInterfaceName, *publicIPAddress, *subnet, *location)
	go createVirtualMachine(*group, *accountName, virtualMachineName, *networkInterface, *publicIPAddress, *location)

	fmt.Printf("\nStarted deployment of virtual machine %s in resource group %s", virtualMachineName, *group.Name)

	time.Sleep(time.Second * 5)

	fmt.Printf("\nNOTE: Since you created a minimal VM without the Azure Linux Agent, the portal will notify you that the deployment failed. After around 50 seconds try connecting to the VM")
	fmt.Printf("\nssh -i path-to-key root@%s\n", *publicIPAddress.DNSSettings.Fqdn)
}
