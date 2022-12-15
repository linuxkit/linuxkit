package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
)

// This program requires that the following environment vars are set:

// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret

const defaultStorageAccountName = "linuxkit"

func runAzureCmd() *cobra.Command {
	var (
		resourceGroupName string
		location          string
		accountName       string
	)

	cmd := &cobra.Command{
		Use:   "azure",
		Short: "launch an Azure instance using an existing image",
		Long: `Launch an Azure instance using an existing image.
		'imagePath' specifies the path (absolute or relative) of a VHD image to be used as the OS image for the VM.

		Relies on the following environment variables:
		
			AZURE_SUBSCRIPTION_ID
			AZURE_TENANT_ID
			AZURE_CLIENT_ID
			AZURE_CLIENT_SECRET

		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run azure [options] imagePath",
		RunE: func(cmd *cobra.Command, args []string) error {
			imagePath := args[0]
			subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
			tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

			clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
			clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

			rand.Seed(time.Now().UTC().UnixNano())
			virtualNetworkName := fmt.Sprintf("linuxkitvirtualnetwork%d", rand.Intn(1000))
			subnetName := fmt.Sprintf("linuxkitsubnet%d", rand.Intn(1000))
			publicIPAddressName := fmt.Sprintf("publicip%d", rand.Intn(1000))
			networkInterfaceName := fmt.Sprintf("networkinterface%d", rand.Intn(1000))
			virtualMachineName := fmt.Sprintf("linuxkitvm%d", rand.Intn(1000))

			initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

			group := createResourceGroup(resourceGroupName, location)
			createStorageAccount(accountName, location, *group)
			uploadVMImage(*group.Name, accountName, imagePath)
			createVirtualNetwork(*group, virtualNetworkName, location)
			subnet := createSubnet(*group, virtualNetworkName, subnetName)
			publicIPAddress := createPublicIPAddress(*group, publicIPAddressName, location)
			networkInterface := createNetworkInterface(*group, networkInterfaceName, *publicIPAddress, *subnet, location)
			go createVirtualMachine(*group, accountName, virtualMachineName, *networkInterface, *publicIPAddress, location)

			fmt.Printf("\nStarted deployment of virtual machine %s in resource group %s", virtualMachineName, *group.Name)

			time.Sleep(time.Second * 5)

			fmt.Printf("\nNOTE: Since you created a minimal VM without the Azure Linux Agent, the portal will notify you that the deployment failed. After around 50 seconds try connecting to the VM")
			fmt.Printf("\nssh -i path-to-key root@%s\n", *publicIPAddress.DNSSettings.Fqdn)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceGroupName, "resourceGroupName", "", "Name of resource group to be used for VM")
	cmd.Flags().StringVar(&location, "location", "westus", "Location of the VM")
	cmd.Flags().StringVar(&accountName, "accountName", defaultStorageAccountName, "Name of the storage account")

	return cmd
}
