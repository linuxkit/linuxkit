package main

import (
	"github.com/spf13/cobra"
)

func pushAzureCmd() *cobra.Command {
	var (
		resourceGroup string
		accountName   string
	)
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "push image to Azure",
		Long: `Push image to Azure.
		First argument specifies the path to a VHD. It will be uploaded to an Azure Storage Account.
		Relies on the following environment variables:
		
			AZURE_SUBSCRIPTION_ID
			AZURE_TENANT_ID
			AZURE_CLIENT_ID
			AZURE_CLIENT_SECRET

		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			subscriptionID := getEnvVarOrExit("AZURE_SUBSCRIPTION_ID")
			tenantID := getEnvVarOrExit("AZURE_TENANT_ID")

			clientID := getEnvVarOrExit("AZURE_CLIENT_ID")
			clientSecret := getEnvVarOrExit("AZURE_CLIENT_SECRET")

			initializeAzureClients(subscriptionID, tenantID, clientID, clientSecret)

			uploadVMImage(resourceGroup, accountName, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceGroup, "resource-group", "", "Name of resource group to be used for VM")
	cmd.Flags().StringVar(&accountName, "storage-account", "", "Name of the storage account")

	return cmd
}
