package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultZone    = "europe-west1-d"
	defaultMachine = "g1-small"
	// Environment variables. Some are non-standard
	zoneVar    = "CLOUDSDK_COMPUTE_ZONE"
	machineVar = "CLOUDSDK_COMPUTE_MACHINE" // non-standard
	keysVar    = "CLOUDSDK_COMPUTE_KEYS"    // non-standard
	projectVar = "CLOUDSDK_CORE_PROJECT"
	bucketVar  = "CLOUDSDK_IMAGE_BUCKET" // non-standard
	familyVar  = "CLOUDSDK_IMAGE_FAMILY" // non-standard
	publicVar  = "CLOUDSDK_IMAGE_PUBLIC" // non-standard
	nameVar    = "CLOUDSDK_IMAGE_NAME"   // non-standard
)

func runGCPCmd() *cobra.Command {
	var (
		name        string
		zoneFlag    string
		machineFlag string
		keysFlag    string
		projectFlag string
		skipCleanup bool
		nestedVirt  bool
		vTPM        bool
		data        string
		dataPath    string
	)

	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "launch a GCP instance",
		Long: `Launch a GCP instance.
		'image' specifies either the name of an already uploaded GCP image,
		or the full path to a image file which will be uploaded before it is run.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run gcp [options] [image]",
		RunE: func(cmd *cobra.Command, args []string) error {
			image := args[0]

			if data != "" && dataPath != "" {
				return errors.New("Cannot specify both -data and -data-file")
			}

			if name == "" {
				name = image
			}

			if dataPath != "" {
				dataB, err := os.ReadFile(dataPath)
				if err != nil {
					return fmt.Errorf("Unable to read metadata file: %v", err)
				}
				data = string(dataB)
			}

			zone := getStringValue(zoneVar, zoneFlag, defaultZone)
			machine := getStringValue(machineVar, machineFlag, defaultMachine)
			keys := getStringValue(keysVar, keysFlag, "")
			project := getStringValue(projectVar, projectFlag, "")

			client, err := NewGCPClient(keys, project)
			if err != nil {
				return fmt.Errorf("Unable to connect to GCP: %v", err)
			}

			if err = client.CreateInstance(name, image, zone, machine, disks, &data, nestedVirt, vTPM, true); err != nil {
				return err
			}

			if err = client.ConnectToInstanceSerialPort(name, zone); err != nil {
				return err
			}

			if !skipCleanup {
				if err = client.DeleteInstance(name, zone, true); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Machine name")
	cmd.Flags().StringVar(&zoneFlag, "zone", defaultZone, "GCP Zone")
	cmd.Flags().StringVar(&machineFlag, "machine", defaultMachine, "GCP Machine Type")
	cmd.Flags().StringVar(&keysFlag, "keys", "", "Path to Service Account JSON key file")
	cmd.Flags().StringVar(&projectFlag, "project", "", "GCP Project Name")
	cmd.Flags().BoolVar(&skipCleanup, "skip-cleanup", false, "Don't remove images or VMs")
	cmd.Flags().BoolVar(&nestedVirt, "nested-virt", false, "Enabled nested virtualization")
	cmd.Flags().BoolVar(&vTPM, "vtpm", false, "Enable vTPM device")

	cmd.Flags().StringVar(&data, "data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	cmd.Flags().StringVar(&dataPath, "data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	return cmd
}
