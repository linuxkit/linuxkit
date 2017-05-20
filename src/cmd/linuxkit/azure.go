package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	simpleStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

var (
	simpleStorageClient     simpleStorage.Client
	groupsClient            resources.GroupsClient
	accountsClient          storage.AccountsClient
	virtualNetworksClient   network.VirtualNetworksClient
	subnetsClient           network.SubnetsClient
	publicIPAddressesClient network.PublicIPAddressesClient
	interfacesClient        network.InterfacesClient
	virtualMachinesClient   compute.VirtualMachinesClient
)

func initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret string) {
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

	groupsClient = resources.NewGroupsClient(subscriptionID)
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(token)

	accountsClient = storage.NewAccountsClient(subscriptionID)
	accountsClient.Authorizer = autorest.NewBearerAuthorizer(token)

	virtualNetworksClient = network.NewVirtualNetworksClient(subscriptionID)
	virtualNetworksClient.Authorizer = autorest.NewBearerAuthorizer(token)

	subnetsClient = network.NewSubnetsClient(subscriptionID)
	subnetsClient.Authorizer = autorest.NewBearerAuthorizer(token)

	publicIPAddressesClient = network.NewPublicIPAddressesClient(subscriptionID)
	publicIPAddressesClient.Authorizer = autorest.NewBearerAuthorizer(token)

	interfacesClient = network.NewInterfacesClient(subscriptionID)
	interfacesClient.Authorizer = autorest.NewBearerAuthorizer(token)

	virtualMachinesClient = compute.NewVirtualMachinesClient(subscriptionID)
	virtualMachinesClient.Authorizer = autorest.NewBearerAuthorizer(token)

}

func getOrCreateResourceGroup(resourceGroupName, location string) *resources.Group {
	var resourceGroup resources.Group
	resourceGroup, err := groupsClient.Get(resourceGroupName)
	if err != nil {
		log.Fatalf("\nError in getting resource group")
	}
	if &resourceGroup != nil {
		return &resourceGroup
	}

	return createResourceGroup(resourceGroupName, location)
}

func createResourceGroup(resourceGroupName, location string) *resources.Group {
	fmt.Printf("\nCreating resource group in %s", location)

	resourceGroupParameters := resources.Group{
		Location: &location,
	}
	group, err := groupsClient.CreateOrUpdate(resourceGroupName, resourceGroupParameters)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to create resource group")
	}

	return &group
}

func createStorageAccount(accountName, location string, resourceGroup resources.Group) *storage.Account {
	fmt.Printf("\nCreating storage account in %s, resource group %s\n", location, *resourceGroup.Name)

	storageAccountCreateParameters := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Name: storage.StandardLRS,
		},
		Location: &location,
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}

	storageChannel, errorChannel := accountsClient.Create(*resourceGroup.Name, accountName, storageAccountCreateParameters, nil)
	var storageAccount storage.Account
	for {
		select {
		case s, ok := <-storageChannel:
			storageAccount = s
			if !ok {
				storageChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if storageChannel == nil && errorChannel == nil {
			break
		}
	}

	return &storageAccount
}

func createVirtualNetwork(resourceGroup resources.Group, virtualNetworkName string, location string) *network.VirtualNetwork {
	fmt.Printf("Creating virtual network in resource group %s, in %s", *resourceGroup.Name, location)

	virtualNetworkParameters := network.VirtualNetwork{
		Location: &location,
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{"10.0.0.0/16"},
			},
		},
	}
	virtualNetworkChannel, errorChannel := virtualNetworksClient.CreateOrUpdate(*resourceGroup.Name, virtualNetworkName, virtualNetworkParameters, nil)
	var virtualNetwork network.VirtualNetwork
	for {
		select {
		case v, ok := <-virtualNetworkChannel:
			virtualNetwork = v
			if !ok {
				virtualNetworkChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if virtualNetworkChannel == nil && errorChannel == nil {
			break
		}
	}

	return &virtualNetwork
}

func createSubnet(resourceGroup resources.Group, virtualNetworkName, subnetName string) *network.Subnet {
	fmt.Printf("\nCreating subnet %s in resource group %s, within virtual network %s", subnetName, *resourceGroup.Name, virtualNetworkName)

	subnetParameters := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/24"),
		},
	}

	subnetChannel, errorChannel := subnetsClient.CreateOrUpdate(*resourceGroup.Name, virtualNetworkName, subnetName, subnetParameters, nil)
	for {
		select {
		case _, ok := <-subnetChannel:
			if !ok {
				subnetChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if subnetChannel == nil && errorChannel == nil {
			break
		}
	}
	subnet, err := subnetsClient.Get(*resourceGroup.Name, virtualNetworkName, subnetName, "")
	if err != nil {
		log.Fatalf("Unable to retrieve subnet")
	}

	return &subnet
}

func createPublicIPAddress(resourceGroup resources.Group, ipName, location string) *network.PublicIPAddress {
	fmt.Printf("\nCreating public IP Address in resource group %s, with name %s", *resourceGroup.Name, ipName)

	ipParameters := network.PublicIPAddress{
		Location: &location,
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			DNSSettings: &network.PublicIPAddressDNSSettings{
				DomainNameLabel: to.StringPtr(fmt.Sprintf("linuxkit%s", ipName)),
			},
		},
	}
	ipAddressChannel, errorChannel := publicIPAddressesClient.CreateOrUpdate(*resourceGroup.Name, ipName, ipParameters, nil)
	for {
		select {
		case _, ok := <-ipAddressChannel:
			if !ok {
				ipAddressChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if ipAddressChannel == nil && errorChannel == nil {
			break
		}
	}
	publicIPAddress, err := publicIPAddressesClient.Get(*resourceGroup.Name, ipName, "")
	if err != nil {
		log.Fatalf("Unable to retrieve public IP address")
	}

	return &publicIPAddress
}

func createNetworkInterface(resourceGroup resources.Group, networkInterfaceName string, publicIPAddress network.PublicIPAddress, subnet network.Subnet, location string) *network.Interface {
	//fmt.Printf("\nCreating network interface in resource group %s, with name %s, in location %s, with public IP %s", *resourceGroup.Name, networkInterfaceName, location, *publicIPAddress.IPAddress)

	networkInterfaceParameters := network.Interface{
		Location: &location,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr(fmt.Sprintf("IPconfig-%s", networkInterfaceName)),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						PublicIPAddress:           &publicIPAddress,
						PrivateIPAllocationMethod: network.Dynamic,
						Subnet: &subnet,
					},
				},
			},
		},
	}
	networkInterfaceChannel, errorChannel := interfacesClient.CreateOrUpdate(*resourceGroup.Name, networkInterfaceName, networkInterfaceParameters, nil)
	for {
		select {
		case _, ok := <-networkInterfaceChannel:
			if !ok {
				networkInterfaceChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if networkInterfaceChannel == nil && errorChannel == nil {
			break
		}
	}

	networkInterface, err := interfacesClient.Get(*resourceGroup.Name, networkInterfaceName, "")
	if err != nil {
		log.Fatalf("Unable to retrieve network interface")
	}
	return &networkInterface
}

// Uploads a file to Azure Storage, to account accountName, in contaiener containerName and blob blobName
func uploadFile(accountName, accountKey, containerName, blobName, filePath string) (blobURL string) {
	simpleStorageClient, err := simpleStorage.NewBasicClient(accountName, accountKey)
	if err != nil {
		log.Fatalf("Unable to create storage client")
	}

	blobClient := simpleStorageClient.GetBlobService()
	options := simpleStorage.CreateContainerOptions{}
	container := blobClient.GetContainerReference(containerName)

	_, err = container.CreateIfNotExists(&options)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Unable to create storage container")
	}

	blob := container.GetBlobReference(blobName)

	file := newFile(filePath)
	defer file.Close()

	reader := bufio.NewReader(file)

	err = blob.CreateBlockBlobFromReader(reader, nil)
	if err != nil {
		fmt.Printf(err.Error())
		log.Fatalf("Unable to create block blob from reader")
	}

	return fmt.Sprintf("You can find the file at https://%s.blob.core.windows.net/%s/%s\n", accountName, containerName, blobName)
}

func newFile(fn string) *os.File {
	fp, err := os.OpenFile(fn, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return fp
}

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		log.Fatalf(fmt.Sprintf("Missing environment variable %s\n", varName))
	}

	return value
}
