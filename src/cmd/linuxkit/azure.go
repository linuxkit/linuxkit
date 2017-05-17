package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"path/filepath"

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

func createStorageAccount(accountName, location string, resourceGroup resources.Group) {
	fmt.Printf("\nCreating storage account in %s, resource group %s\n", location, *resourceGroup.Name)

	storageAccountCreateParameters := storage.AccountCreateParameters{
		Sku: &storage.Sku{
			Name: storage.StandardLRS,
		},
		Location: &location,
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}

	storageChannel, errorChannel := accountsClient.Create(*resourceGroup.Name, accountName, storageAccountCreateParameters, nil)
	for {
		select {
		case _, ok := <-storageChannel:
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
}

func uploadVMImage(resourceGroupName string, accountName string, imagePath string) {
	accountKeys, err := accountsClient.ListKeys(resourceGroupName, accountName)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to retrieve storage account key")
	}

	keys := *(accountKeys.Keys)

	absolutePath, err := filepath.Abs(imagePath)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to get absolute path")
	}

	directory, image := filepath.Split(absolutePath)

	dockerMount := fmt.Sprintf("%s:/vhds", directory)
	storageAccountNameArg := fmt.Sprintf("STORAGE_ACCOUNT_NAME=%s", accountName)
	storageAccountKeyArg := fmt.Sprintf("STORAGE_ACCOUNT_KEY=%s", *keys[0].Value)
	vhdPath := fmt.Sprintf("VHD_PATH=/vhds/%s", image)

	output, err := exec.Command("docker", "run", "-v", dockerMount, "-e", vhdPath, "-e", storageAccountNameArg, "-e", storageAccountKeyArg, "radumatei/azure-vhd-upload:alpine").CombinedOutput()

	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Println(string(output))

	fmt.Printf("OS Image uploaded at https://%s.blob.core.windows.net/linuxkitcontainer/linuxkitimage.vhd\n", accountName)
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
				DomainNameLabel: to.StringPtr(ipName),
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
	time.Sleep(time.Second * 5)
	publicIPAddress, err := publicIPAddressesClient.Get(*resourceGroup.Name, ipName, "")
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalf("Unable to retrieve public IP address")
	}
	return &publicIPAddress
}

func createNetworkInterface(resourceGroup resources.Group, networkInterfaceName string, publicIPAddress network.PublicIPAddress, subnet network.Subnet, location string) *network.Interface {

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

func setVirtualMachineParameters(storageAccountName string, networkInterfaceID, location string) compute.VirtualMachine {
	return compute.VirtualMachine{
		Location: &location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.StandardDS1,
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr("linuxkit"),
				AdminUsername: to.StringPtr("dummyusername"),
				AdminPassword: to.StringPtr("DummyPassword!123"),
			},
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr("osDisk"),
					OsType:       compute.Linux,
					Caching:      compute.ReadWrite,
					CreateOption: compute.FromImage,
					Image: &compute.VirtualHardDisk{
						URI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net/linuxkitcontainer/linuxkitimage.vhd", storageAccountName)),
					},
					Vhd: &compute.VirtualHardDisk{
						URI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net/data/data.vhd", storageAccountName)),
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: &networkInterfaceID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}
}

func createVirtualMachine(resourceGroup resources.Group, storageAccountName string, virtualMachineName string, networkInterface network.Interface, publicIPAddress network.PublicIPAddress, location string) {
	fmt.Printf("\nCreating virtual machine in resource group %s, with name %s, in location %s", *resourceGroup.Name, virtualMachineName, location)

	virtualMachineParameters := setVirtualMachineParameters(storageAccountName, *networkInterface.ID, location)
	virtualMachineChannel, errorChannel := virtualMachinesClient.CreateOrUpdate(*resourceGroup.Name, virtualMachineName, virtualMachineParameters, nil)
	for {
		select {
		case _, ok := <-virtualMachineChannel:
			if !ok {
				virtualMachineChannel = nil
			}
		case _, ok := <-errorChannel:
			if !ok {
				errorChannel = nil
			}
		}
		if virtualMachineChannel == nil && errorChannel == nil {
			break
		}
	}

}

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		log.Fatalf(fmt.Sprintf("Missing environment variable %s\n", varName))
	}

	return value
}
