package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultScalewayVolumeSize = 10 // GB

func pushScalewayCmd() *cobra.Command {
	var (
		nameFlag           string
		accessKeyFlag      string
		secretKeyFlag      string
		sshKeyFlag         string
		instanceIDFlag     string
		deviceNameFlag     string
		volumeSizeFlag     int
		zoneFlag           string
		organizationIDFlag string
		noCleanFlag        bool
	)
	cmd := &cobra.Command{
		Use:   "scaleway",
		Short: "push image to Scaleway",
		Long: `Push image to Scaleway.
		First argument specifies the path to an EFI ISO image. It will be copied to a new Scaleway instance in order to create a Scaeway image out of it.
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			name := getStringValue(scalewayNameVar, nameFlag, "")
			accessKey := getStringValue(accessKeyVar, accessKeyFlag, "")
			secretKey := getStringValue(secretKeyVar, secretKeyFlag, "")
			sshKeyFile := getStringValue(sshKeyVar, sshKeyFlag, "")
			instanceID := getStringValue(instanceIDVar, instanceIDFlag, "")
			deviceName := getStringValue(deviceNameVar, deviceNameFlag, "")
			volumeSize := getIntValue(volumeSizeVar, volumeSizeFlag, 0)
			zone := getStringValue(zoneVar, zoneFlag, defaultScalewayZone)
			organizationID := getStringValue(organizationIDVar, organizationIDFlag, "")

			const suffix = ".iso"
			if name == "" {
				name = strings.TrimSuffix(path, suffix)
				name = filepath.Base(name)
			}

			client, err := NewScalewayClient(accessKey, secretKey, zone, organizationID)
			if err != nil {
				return fmt.Errorf("Unable to connect to Scaleway: %v", err)
			}

			// if volume size not set, try to calculate it from file size
			if volumeSize == 0 {
				if fi, err := os.Stat(path); err == nil {
					volumeSize = int(math.Ceil(float64(fi.Size()) / 1000000000)) // / 1 GB
				} else {
					// fallback to default
					log.Warnf("Unable to calculate volume size, using default of %d GB: %v", defaultScalewayVolumeSize, err)
					volumeSize = defaultScalewayVolumeSize
				}
			}

			// if no instanceID is provided, we create the instance
			if instanceID == "" {
				instanceID, err = client.CreateInstance(volumeSize)
				if err != nil {
					return fmt.Errorf("Error creating a Scaleway instance: %v", err)
				}

				err = client.BootInstanceAndWait(instanceID)
				if err != nil {
					return fmt.Errorf("Error booting instance: %v", err)
				}
			}

			volumeID, err := client.GetSecondVolumeID(instanceID)
			if err != nil {
				return fmt.Errorf("Error retrieving second volume ID: %v", err)
			}

			err = client.CopyImageToInstance(instanceID, path, sshKeyFile)
			if err != nil {
				return fmt.Errorf("Error copying ISO file to Scaleway's instance: %v", err)
			}

			err = client.WriteImageToVolume(instanceID, deviceName)
			if err != nil {
				return fmt.Errorf("Error writing ISO file to additional volume: %v", err)
			}

			err = client.TerminateInstance(instanceID)
			if err != nil {
				return fmt.Errorf("Error terminating Scaleway's instance: %v", err)
			}

			err = client.CreateScalewayImage(instanceID, volumeID, name)
			if err != nil {
				return fmt.Errorf("Error creating Scaleway image: %v", err)
			}

			if !noCleanFlag {
				err = client.DeleteInstanceAndVolumes(instanceID)
				if err != nil {
					return fmt.Errorf("Error deleting Scaleway instance and volumes: %v", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&nameFlag, "img-name", "", "Overrides the name used to identify the image name in Scaleway's images. Defaults to the base of 'path' with the '.iso' suffix removed")
	cmd.Flags().StringVar(&accessKeyFlag, "access-key", "", "Access Key to connect to Scaleway API")
	cmd.Flags().StringVar(&secretKeyFlag, "secret-key", "", "Secret Key to connect to Scaleway API")
	cmd.Flags().StringVar(&sshKeyFlag, "ssh-key", os.Getenv("HOME")+"/.ssh/id_rsa", "SSH key file")
	cmd.Flags().StringVar(&instanceIDFlag, "instance-id", "", "Instance ID of a running Scaleway instance, with a second volume.")
	cmd.Flags().StringVar(&deviceNameFlag, "device-name", "/dev/vdb", "Device name on which the image will be copied")
	cmd.Flags().IntVar(&volumeSizeFlag, "volume-size", 0, "Size of the volume to use (in GB). Defaults to size of the ISO file rounded up to GB")
	cmd.Flags().StringVar(&zoneFlag, "zone", defaultScalewayZone, "Select Scaleway zone")
	cmd.Flags().StringVar(&organizationIDFlag, "organization-id", "", "Select Scaleway's organization ID")
	cmd.Flags().BoolVar(&noCleanFlag, "no-clean", false, "Do not remove temporary instance and volumes")

	return cmd
}
