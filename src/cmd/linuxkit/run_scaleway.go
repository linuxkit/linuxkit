package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultScalewayInstanceType = "DEV1-S"
	defaultScalewayZone         = "par1"

	scalewayNameVar   = "SCW_IMAGE_NAME" // non-standard
	accessKeyVar      = "SCW_ACCESS_KEY"
	secretKeyVar      = "SCW_SECRET_KEY"
	sshKeyVar         = "SCW_SSH_KEY_FILE" // non-standard
	instanceIDVar     = "SCW_INSTANCE_ID"  // non-standard
	deviceNameVar     = "SCW_DEVICE_NAME"  // non-standard
	volumeSizeVar     = "SCW_VOLUME_SIZE"  // non-standard
	scwZoneVar        = "SCW_DEFAULT_ZONE"
	organizationIDVar = "SCW_DEFAULT_ORGANIZATION_ID"

	instanceTypeVar = "SCW_RUN_TYPE" // non-standard
)

func runScalewayCmd() *cobra.Command {
	var (
		instanceTypeFlag   string
		instanceNameFlag   string
		accessKeyFlag      string
		secretKeyFlag      string
		zoneFlag           string
		organizationIDFlag string
		cleanFlag          bool
		noAttachFlag       bool
	)

	cmd := &cobra.Command{
		Use:   "scaleway",
		Short: "launch a scaleway instance",
		Long: `Launch an Scaleway instance using an existing image.
		'name' is the name of a Scaleway image that has already been uploaded using 'linuxkit push'.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run scaleway [options] [name]",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			instanceType := getStringValue(instanceTypeVar, instanceTypeFlag, defaultScalewayInstanceType)
			instanceName := getStringValue("", instanceNameFlag, name)
			accessKey := getStringValue(accessKeyVar, accessKeyFlag, "")
			secretKey := getStringValue(secretKeyVar, secretKeyFlag, "")
			zone := getStringValue(scwZoneVar, zoneFlag, defaultScalewayZone)
			organizationID := getStringValue(organizationIDVar, organizationIDFlag, "")

			client, err := NewScalewayClient(accessKey, secretKey, zone, organizationID)
			if err != nil {
				log.Fatalf("Unable to connect to Scaleway: %v", err)
			}

			instanceID, err := client.CreateLinuxkitInstance(instanceName, name, instanceType)
			if err != nil {
				log.Fatalf("Unable to create Scaleway instance: %v", err)
			}

			err = client.BootInstance(instanceID)
			if err != nil {
				log.Fatalf("Unable to boot Scaleway instance: %v", err)
			}

			if !noAttachFlag {
				err = client.ConnectSerialPort(instanceID)
				if err != nil {
					log.Fatalf("Unable to connect to serial port: %v", err)
				}
			}

			if cleanFlag {
				err = client.TerminateInstance(instanceID)
				if err != nil {
					log.Fatalf("Unable to stop instance: %v", err)
				}

				err = client.DeleteInstanceAndVolumes(instanceID)
				if err != nil {
					log.Fatalf("Unable to delete instance: %v", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&instanceTypeFlag, "instance-type", defaultScalewayInstanceType, "Scaleway instance type")
	cmd.Flags().StringVar(&instanceNameFlag, "instance-name", "linuxkit", "Name of the create instance, default to the image name")
	cmd.Flags().StringVar(&accessKeyFlag, "access-key", "", "Access Key to connect to Scaleway API")
	cmd.Flags().StringVar(&secretKeyFlag, "secret-key", "", "Secret Key to connect to Scaleway API")
	cmd.Flags().StringVar(&zoneFlag, "zone", defaultScalewayZone, "Select Scaleway zone")
	cmd.Flags().StringVar(&organizationIDFlag, "organization-id", "", "Select Scaleway's organization ID")
	cmd.Flags().BoolVar(&cleanFlag, "clean", false, "Remove instance")
	cmd.Flags().BoolVar(&noAttachFlag, "no-attach", false, "Don't attach to serial port, you will have to connect to instance manually")

	return cmd
}
