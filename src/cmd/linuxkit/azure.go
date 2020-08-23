package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	simpleStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Microsoft/azure-vhd-utils/upload"
	uploadMetaData "github.com/Microsoft/azure-vhd-utils/upload/metadata"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/common"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/diskstream"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/validator"
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
	group, err := groupsClient.CreateOrUpdate(context.Background(), resourceGroupName, resourceGroupParameters)
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
		Location:                          &location,
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
	}
	for {
		if _, err := accountsClient.Create(context.Background(), *resourceGroup.Name, accountName, storageAccountCreateParameters); err != nil {
			time.Sleep(time.Second * 5)
		}
	}
}

func uploadVMImage(resourceGroupName string, accountName string, imagePath string) {

	const PageBlobPageSize int64 = 2 * 1024 * 1024
	parallelism := 8 * runtime.NumCPU()

	accountKeys, err := accountsClient.ListKeys(context.Background(), resourceGroupName, accountName, "")
	if err != nil {
		log.Fatalf("Unable to retrieve storage account key: %v", err)
	}

	keys := *(accountKeys.Keys)

	absolutePath, err := filepath.Abs(imagePath)
	if err != nil {
		log.Fatalf("Unable to get absolute path: %v", err)
	}

	// directory, image := filepath.Split(absolutePath)

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
	container := blobServiceClient.GetContainerReference(defaultStorageContainerName)
	resp, err := container.CreateIfNotExists(&simpleStorage.CreateContainerOptions{Access: simpleStorage.ContainerAccessTypePrivate})
	if err != nil || !resp {
		log.Fatalf("Unable to create or retrieve container: %v", err)
	}

	localMetaData := getLocalVHDMetaData(absolutePath)

	blob := container.GetBlobReference(defaultStorageBlobName)
	err = blob.PutPageBlob(nil)
	if err != nil {
		log.Fatalf("Unable to create VHD blob: %v", err)
	}

	m, _ := localMetaData.ToMap()
	blob.Metadata = m

	if err = blob.SetMetadata(nil); err != nil {
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
	if future, err := virtualNetworksClient.CreateOrUpdate(context.Background(), *resourceGroup.Name, virtualNetworkName, virtualNetworkParameters); err == nil {
		if virtualNetwork, err := future.Result(virtualNetworksClient); err != nil {
			return &virtualNetwork
		}
	}
	return nil
}

func createSubnet(resourceGroup resources.Group, virtualNetworkName, subnetName string) *network.Subnet {
	fmt.Printf("Creating subnet %s in resource group %s, within virtual network %s\n", subnetName, *resourceGroup.Name, virtualNetworkName)

	subnetParameters := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr(defaultSubnetAddressPrefix),
		},
	}

	var err error
	if future, err := subnetsClient.CreateOrUpdate(context.Background(), *resourceGroup.Name, virtualNetworkName, subnetName, subnetParameters); err == nil {
		if subnet, err := future.Result(subnetsClient); err == nil {
			return &subnet
		}
	}

	log.Fatalf("Unable to retrieve subnet: %v", err)
	return nil
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

	var err error
	if future, err := publicIPAddressesClient.CreateOrUpdate(context.Background(), *resourceGroup.Name, ipName, ipParameters); err == nil {
		if publicIPAddress, err := future.Result(publicIPAddressesClient); err == nil {
			return &publicIPAddress
		}
	}

	log.Fatalf("Unable to retrieve public IP address: %v", err)
	return nil
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
						Subnet:                    &subnet,
					},
				},
			},
		},
	}

	var err error
	if future, err := interfacesClient.CreateOrUpdate(context.Background(), *resourceGroup.Name, networkInterfaceName, networkInterfaceParameters); err == nil {
		if networkInterface, err := future.Result(interfacesClient); err == nil {
			return &networkInterface
		}
	}
	log.Fatalf("Unable to retrieve network interface: %v", err)
	return nil
}

func setVirtualMachineParameters(storageAccountName string, networkInterfaceID, location string) compute.VirtualMachine {
	return compute.VirtualMachine{
		Location: &location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypesStandardDS1,
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
					Caching:      compute.CachingTypesReadWrite,
					CreateOption: compute.DiskCreateOptionTypesFromImage,
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
	var err error
	if future, err := virtualMachinesClient.CreateOrUpdate(context.Background(), *resourceGroup.Name, virtualMachineName, virtualMachineParameters); err == nil {
		if _, err := future.Result(virtualMachinesClient); err == nil {
			return
		}
	}
	panic(err)
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
		blob := client.GetContainerReference(containerName).GetBlobReference(blobName)
		blob.Properties.ContentMD5 = base64.StdEncoding.EncodeToString(vhdMetaData.FileMetaData.MD5Hash)
		if err := blob.SetProperties(nil); err != nil {
			log.Fatalf("Unable to set blob properties: %v", err)
		}
	}
}
