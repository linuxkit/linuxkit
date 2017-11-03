package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	simpleStorage "github.com/radu-matei/azure-sdk-for-go/storage"
	"github.com/radu-matei/azure-vhd-utils/upload"
	uploadMetaData "github.com/radu-matei/azure-vhd-utils/upload/metadata"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/diskstream"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/validator"
)

const (
	defaultStorageContainerName = "linuxkitcontainer"
	defaultStorageBlobName      = "linuxkitimage.vhd"

	defaultVMStorageContainerName = "data"
	defaultVMStorageBlobName      = "data.vhd"

	defaultVirtualNetworkAddressPrefix = "10.0.0.0/16"
	defaultSubnetAddressPrefix         = "10.0.0.0/24"

	// These values are only provided so the deployment gets validated
	// Since there is currently no Azure Linux Agent, these values
	// will not be enforced on the VM

	defaultComputerName = "linuxkit"
	unusedAdminUsername = "unusedUserName"
	unusedPassword      = "UnusedPassword!123"
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

	defaultActiveDirectoryEndpoint = azure.PublicCloud.ActiveDirectoryEndpoint
	defaultResourceManagerEndpoint = azure.PublicCloud.ResourceManagerEndpoint
)

func initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret string) {
	oAuthConfig, err := adal.NewOAuthConfig(defaultActiveDirectoryEndpoint, tenantID)
	if err != nil {
		log.Fatalf("Cannot get oAuth configuration: %v", err)
	}

	token, err := adal.NewServicePrincipalToken(*oAuthConfig, clientID, clientSecret, defaultResourceManagerEndpoint)
	if err != nil {
		log.Fatalf("Cannot get service principal token: %v", err)
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

func createResourceGroup(resourceGroupName, location string) *resources.Group {
	fmt.Printf("Creating resource group in %s\n", location)

	resourceGroupParameters := resources.Group{
		Location: &location,
	}
	group, err := groupsClient.CreateOrUpdate(resourceGroupName, resourceGroupParameters)
	if err != nil {
		log.Fatalf("Unable to create resource group: %v", err)
	}

	return &group
}

func createStorageAccount(accountName, location string, resourceGroup resources.Group) {
	fmt.Printf("Creating storage account in %s, resource group %s\n", location, *resourceGroup.Name)

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

	time.Sleep(time.Second * 5)
}

func uploadVMImage(resourceGroupName string, accountName string, imagePath string) {

	const PageBlobPageSize int64 = 2 * 1024 * 1024
	parallelism := 8 * runtime.NumCPU()

	accountKeys, err := accountsClient.ListKeys(resourceGroupName, accountName)
	if err != nil {
		log.Fatalf("Unable to retrieve storage account key: %v", err)
	}

	keys := *(accountKeys.Keys)

	absolutePath, err := filepath.Abs(imagePath)
	if err != nil {
		log.Fatalf("Unable to get absolute path: %v", err)
	}

	//directory, image := filepath.Split(absolutePath)

	ensureVHDSanity(absolutePath)

	diskStream, err := diskstream.CreateNewDiskStream(absolutePath)
	if err != nil {
		log.Fatalf("Unable to create disk stream for VHD: %v", err)
	}
	defer diskStream.Close()

	simpleStorageClient, err = simpleStorage.NewBasicClient(accountName, *keys[0].Value)
	if err != nil {
		log.Fatalf("Unable to create simple storage client: %v", err)
	}

	blobServiceClient := simpleStorageClient.GetBlobService()
	_, err = blobServiceClient.CreateContainerIfNotExists(defaultStorageContainerName, simpleStorage.ContainerAccessTypePrivate)
	if err != nil {
		log.Fatalf("Unable to create or retrieve container: %v", err)
	}

	localMetaData := getLocalVHDMetaData(absolutePath)

	err = blobServiceClient.PutPageBlob(defaultStorageContainerName, defaultStorageBlobName, diskStream.GetSize(), nil)
	if err != nil {
		log.Fatalf("Unable to create VHD blob: %v", err)
	}

	m, _ := localMetaData.ToMap()
	err = blobServiceClient.SetBlobMetadata(defaultStorageContainerName, defaultStorageBlobName, m, make(map[string]string))
	if err != nil {
		log.Fatalf("Unable to set blob metatada: %v", err)
	}

	var rangesToSkip []*common.IndexRange
	uploadableRanges, err := upload.LocateUploadableRanges(diskStream, rangesToSkip, PageBlobPageSize)
	if err != nil {
		log.Fatalf("Unable to locate uploadable ranges: %v", err)
	}

	uploadableRanges, err = upload.DetectEmptyRanges(diskStream, uploadableRanges)
	if err != nil {
		log.Fatalf("Unable to detect empty blob ranges: %v", err)
	}

	cxt := &upload.DiskUploadContext{
		VhdStream:             diskStream,
		UploadableRanges:      uploadableRanges,
		AlreadyProcessedBytes: common.TotalRangeLength(rangesToSkip),
		BlobServiceClient:     blobServiceClient,
		ContainerName:         defaultStorageContainerName,
		BlobName:              defaultStorageBlobName,
		Parallelism:           parallelism,
		Resume:                false,
		MD5Hash:               localMetaData.FileMetaData.MD5Hash,
	}

	err = upload.Upload(cxt)
	if err != nil {
		log.Fatalf("Unable to upload VHD: %v", err)
	}

	setBlobMD5Hash(blobServiceClient, defaultStorageContainerName, defaultStorageBlobName, localMetaData)

}

func createVirtualNetwork(resourceGroup resources.Group, virtualNetworkName string, location string) *network.VirtualNetwork {
	fmt.Printf("Creating virtual network in resource group %s, in %s", *resourceGroup.Name, location)

	virtualNetworkParameters := network.VirtualNetwork{
		Location: &location,
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{defaultVirtualNetworkAddressPrefix},
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
	fmt.Printf("Creating subnet %s in resource group %s, within virtual network %s\n", subnetName, *resourceGroup.Name, virtualNetworkName)

	subnetParameters := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr(defaultSubnetAddressPrefix),
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
		log.Fatalf("Unable to retrieve subnet: %v", err)
	}

	return &subnet
}

func createPublicIPAddress(resourceGroup resources.Group, ipName, location string) *network.PublicIPAddress {
	fmt.Printf("Creating public IP Address in resource group %s, with name %s\n", *resourceGroup.Name, ipName)

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
		log.Fatalf("Unable to retrieve public IP address: %v", err)
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
		log.Fatalf("Unable to retrieve network interface: %v", err)
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
			// This is only for deployment validation.
			// The values here will not be usable by anyone

			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr(defaultComputerName),
				AdminUsername: to.StringPtr(unusedAdminUsername),
				AdminPassword: to.StringPtr(unusedPassword),
			},
			StorageProfile: &compute.StorageProfile{
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr("osDisk"),
					OsType:       compute.Linux,
					Caching:      compute.ReadWrite,
					CreateOption: compute.FromImage,
					Image: &compute.VirtualHardDisk{
						URI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccountName, defaultStorageContainerName, defaultStorageBlobName)),
					},
					Vhd: &compute.VirtualHardDisk{
						URI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccountName, defaultVMStorageContainerName, defaultVMStorageBlobName)),
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
			DiagnosticsProfile: &compute.DiagnosticsProfile{
				BootDiagnostics: &compute.BootDiagnostics{
					Enabled:    to.BoolPtr(true),
					StorageURI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName)),
				},
			},
		},
	}
}

func createVirtualMachine(resourceGroup resources.Group, storageAccountName string, virtualMachineName string, networkInterface network.Interface, publicIPAddress network.PublicIPAddress, location string) {
	fmt.Printf("Creating virtual machine in resource group %s, with name %s, in location %s\n", *resourceGroup.Name, virtualMachineName, location)

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
		log.Fatalf("Missing environment variable %s\n", varName)
	}

	return value
}

func ensureVHDSanity(localVHDPath string) {
	if err := validator.ValidateVhd(localVHDPath); err != nil {
		log.Fatalf("Unable to validate VHD: %v", err)
	}

	if err := validator.ValidateVhdSize(localVHDPath); err != nil {
		log.Fatalf("Unable to validate VHD size: %v", err)
	}
}

func getLocalVHDMetaData(localVHDPath string) *uploadMetaData.MetaData {
	localMetaData, err := uploadMetaData.NewMetaDataFromLocalVHD(localVHDPath)
	if err != nil {
		log.Fatalf("Unable to get VHD metadata: %v", err)
	}
	return localMetaData
}

func setBlobMD5Hash(client simpleStorage.BlobStorageClient, containerName, blobName string, vhdMetaData *uploadMetaData.MetaData) {
	if vhdMetaData.FileMetaData.MD5Hash != nil {
		blobHeaders := simpleStorage.BlobHeaders{
			ContentMD5: base64.StdEncoding.EncodeToString(vhdMetaData.FileMetaData.MD5Hash),
		}
		if err := client.SetBlobProperties(containerName, blobName, blobHeaders); err != nil {
			log.Fatalf("Unable to set blob properties: %v", err)
		}
	}
}
