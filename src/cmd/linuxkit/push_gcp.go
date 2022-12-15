package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func pushGCPCmd() *cobra.Command {
	var (
		keysFlag    string
		projectFlag string
		bucketFlag  string
		publicFlag  bool
		familyFlag  string
		nameFlag    string
		nestedVirt  bool
		uefi        bool
	)
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "push image to GCP",
		Long: `Push image to GCP.
		First argument specifies the path to a disk file.
		It will be uploaded to GCS and GCP VM image will be created from it.
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			keys := getStringValue(keysVar, keysFlag, "")
			project := getStringValue(projectVar, projectFlag, "")
			bucket := getStringValue(bucketVar, bucketFlag, "")
			public := getBoolValue(publicVar, publicFlag)
			family := getStringValue(familyVar, familyFlag, "")
			name := getStringValue(nameVar, nameFlag, "")

			const suffix = ".img.tar.gz"
			if name == "" {
				name = strings.TrimSuffix(path, suffix)
				name = filepath.Base(name)
			}

			client, err := NewGCPClient(keys, project)
			if err != nil {
				return fmt.Errorf("Unable to connect to GCP: %v", err)
			}

			if bucket == "" {
				return fmt.Errorf("Please specify the bucket to use")
			}

			err = client.UploadFile(path, name+suffix, bucket, public)
			if err != nil {
				return fmt.Errorf("Error copying to Google Storage: %v", err)
			}
			err = client.CreateImage(name, "https://storage.googleapis.com/"+bucket+"/"+name+suffix, family, nestedVirt, uefi, true)
			if err != nil {
				return fmt.Errorf("Error creating Google Compute Image: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&keysFlag, "keys", "", "Path to Service Account JSON key file")
	cmd.Flags().StringVar(&projectFlag, "project", "", "GCP Project Name")
	cmd.Flags().StringVar(&bucketFlag, "bucket", "", "GCS Bucket to upload to. *Required*")
	cmd.Flags().BoolVar(&publicFlag, "public", false, "Select if file on GCS should be public. *Optional*")
	cmd.Flags().StringVar(&familyFlag, "family", "", "GCP Image Family. A group of images where the family name points to the most recent image. *Optional*")
	cmd.Flags().StringVar(&nameFlag, "img-name", "", "Overrides the name used to identify the file in Google Storage and the VM image. Defaults to the base of 'path' with the '.img.tar.gz' suffix removed")
	cmd.Flags().BoolVar(&nestedVirt, "nested-virt", false, "Enabled nested virtualization for the image")
	cmd.Flags().BoolVar(&uefi, "uefi-compatible", false, "Enable UEFI_COMPATIBLE feature for the image, required to enable vTPM.")

	return cmd
}
