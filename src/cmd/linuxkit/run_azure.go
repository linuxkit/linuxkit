package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

// This program requires that the following environment vars are set:

// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret

const ()

var (
	vmClient compute.VirtualMachinesClient
)

// Process the run arguments and execute run
func runAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run azure [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("GCP image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
	tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

	clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
	clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

	resourceGroup := flags.String("resourceGroup", "", "Name of resource group to be used for VM")
	vmName := flags.String("vmName", "default-linuxkit-vm", "Name of the Azure VM")
	vhdURI := flags.String("vhdUri", "", "Address of the VHD to be used as image for VM")
	location := flags.String("location", "westus", "Location of the VM")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	vmClient := getVMClient(subscriptionID, tenantID, clientID, clientSecret)
	vmClient.CreateOrUpdate(*resourceGroup, *vmName, setVMParameters(*location, *vhdURI), nil)

}

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		fmt.Printf("Missing environment variable %s\n", varName)
		os.Exit(1)
	}

	return value
}

func getVMClient(subscriptionID, tenantID, clientID, clientSecret string) compute.VirtualMachinesClient {
	oAuthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Cannot get oAuth configuration")
	}

	token, err := adal.NewServicePrincipalToken(*oAuthConfig, clientID, clientSecret, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Cannot get service principal token")
	}

	vmClient = compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = autorest.NewBearerAuthorizer(token)

	return vmClient
}

func setVMParameters(location, vhdURI string) compute.VirtualMachine {
	return compute.VirtualMachine{
		Location: &location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr("os-disk"),
					CreateOption: compute.FromImage,
					OsType:       compute.Linux,
					Image: &compute.VirtualHardDisk{
						URI: &vhdURI,
					},
				},
			},
		},
	}
}

/*
func setVMParameters(location, vhdURI string) compute.VirtualMachine {
	return compute.VirtualMachine{
		Location: &location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					Name: to.StringPtr("os-disk"),
					Image: &compute.VirtualHardDisk{
						URI: &vhdURI,
					},
				},
			},
		},
	}
}
*/
