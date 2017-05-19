package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
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
	vmClient             compute.VirtualMachinesClient
	groupsClient         resources.GroupsClient
	storageAccountClient storage.AccountsClient
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
	vhdURI := flags.String("vhdURI", "", "Address of the VHD to be used as image for VM")
	location := flags.String("location", "westus", "Location of the VM")
	//storageAccountName := flags.String("storageAccountName", "linuxkit-storage", "Name of the storage account")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	setAzureClients(subscriptionID, tenantID, clientID, clientSecret)
	//createAdditionalResources(*resourceGroup, *location, *storageAccountName)

	resultChannel, errChannel := vmClient.CreateOrUpdate(*resourceGroup, *vmName, setVMParameters(*location, *vhdURI), nil)

	for {
		select {
		case resultVM, ok := <-resultChannel:
			fmt.Println("result", resultVM, ok)
			if !ok {
				resultChannel = nil
			}
		case error, ok := <-errChannel:
			fmt.Println("error", error, ok)
			if !ok {
				errChannel = nil
			}
		}

		if resultChannel == nil && errChannel == nil {
			fmt.Println("done reading from channels")
			break
		}
	}

}

func createAdditionalResources(resourceGroupName, location, storageAccountName string) (*resources.Group, *storage.Account) {
	fmt.Println("Creating additional resources...")
	fmt.Printf("Creating resource group in %s\n", location)

	resourceGroupOptions := resources.Group{
		Location: &location,
	}

	resourceGroup, err := groupsClient.CreateOrUpdate(resourceGroupName, resourceGroupOptions)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to create resource group")
	}

	storageAccountParameters := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Name: storage.StandardLRS,
		},
		Location: &location,
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}

	/*storageChannel, errorChannel := */
	storageAccountClient.Create(resourceGroupName, storageAccountName, storageAccountParameters, nil)

	/*
		for {
			select {
			case resultStorage, ok := <-storageChannel:
				fmt.Println("result", resultStorage, ok)
				if !ok {
					storageChannel = nil
				}
			case err, ok := <-errorChannel:
				fmt.Println("error", err, ok)
				if !ok {
					errorChannel = nil
				}
			}

			if storageChannel == nil && errorChannel == nil {
				break
			}
		}
	*/
	return &resourceGroup, nil
}

func createWinServer(subscriptionID string) compute.VirtualMachine {
	i := 10
	var i32 int32
	i32 = int32(i)
	return compute.VirtualMachine{
		Location: to.StringPtr("westus"),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: to.StringPtr("MicrosoftWindowsServerEssentials"),
					Offer:     to.StringPtr("WindowsServerEssentials"),
					Sku:       to.StringPtr("WindowsServerEssentials"),
					Version:   to.StringPtr("latest"),
					ID:        to.StringPtr(fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/westus/publishers/MicrosoftWindowsServerEssentials/ArtifactTypes/vmimage/offers/WindowsServerEssentials/skus/WindowsServerEssentials/versions/latest", subscriptionID)),
				},
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr("os-disk"),
					OsType:       compute.Windows,
					CreateOption: compute.FromImage,
					DiskSizeGB:   &i32,
				},
			},
		},
	}
}

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		log.Fatalf(fmt.Sprintf("Missing environment variable %s\n", varName))
	}

	return value
}

func setAzureClients(subscriptionID, tenantID, clientID, clientSecret string) {
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

	groupsClient := resources.NewGroupsClient(subscriptionID)
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(token)

	storageAccountClient := storage.NewAccountsClient(subscriptionID)
	storageAccountClient.Authorizer = autorest.NewBearerAuthorizer(token)
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

//Testing the authorization. Will be removed
func listVMs() {
	fmt.Println("List VMs in subscription...")
	list, err := vmClient.ListAll()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to list VMs")
	}

	if list.Value != nil && len(*list.Value) > 0 {
		fmt.Println("VMs in subscription")
		for _, vm := range *list.Value {
			printVM(vm)
		}
	} else {
		fmt.Println("There are no VMs in this subscription")
	}
}

func printVM(vm compute.VirtualMachine) {
	tags := "\n"
	if vm.Tags == nil {
		tags += "\t\tNo tags yet\n"
	} else {
		for k, v := range *vm.Tags {
			tags += fmt.Sprintf("\t\t%s = %s\n", k, *v)
		}
	}
	fmt.Printf("Virtual machine '%s'\n", *vm.Name)
	elements := map[string]interface{}{
		"ID":       *vm.ID,
		"Type":     *vm.Type,
		"Location": *vm.Location,
		"Tags":     tags}
	for k, v := range elements {
		fmt.Printf("\t%s: %s\n", k, v)
	}
}
