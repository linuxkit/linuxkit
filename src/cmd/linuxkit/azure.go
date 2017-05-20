package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	simpleStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

var (
	simpleStorageClient   simpleStorage.Client
	groupsClient          resources.GroupsClient
	accountsClient        storage.AccountsClient
	virtualMachinesClient compute.VirtualMachinesClient
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
	fmt.Printf("Creating resource group in %s", location)

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
	fmt.Printf("Creating storage account in %s, resource group %s\n", location, *resourceGroup.Name)

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

	return &storageAccount
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
