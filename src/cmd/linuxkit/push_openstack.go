package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

func createOpenStackImage(filePath string, imageName string, client *gophercloud.ServiceClient) {
	// Image formats that are supported by both LinuxKit and OpenStack Glance V2
	formats := []string{"ami", "vhd", "vhdx", "vmdk", "raw", "qcow2", "iso"}

	// Find extension of the filename and remove the leading stop
	fileExtension := strings.Replace(path.Ext(filePath), ".", "", -1)
	fileName := strings.TrimSuffix(path.Base(filePath), filepath.Ext(filePath))
	// Check for Supported extension
	var supportedExtension bool
	supportedExtension = false
	for i := 0; i < len(formats); i++ {
		if strings.ContainsAny(fileExtension, formats[i]) {
			supportedExtension = true
		}
	}

	if !supportedExtension {
		log.Fatalf("Extension [%s] is not supported", fileExtension)
	}

	if imageName == "" {
		imageName = fileName
	}

	imageOpts := images.CreateOpts{
		Name:            imageName,
		ContainerFormat: "bare",
		DiskFormat:      fileExtension,
	}
	image, err := images.Create(client, imageOpts).Extract()
	if err != nil {
		log.Fatalf("Error creating image: %s", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Can't read image file: %s", err)
	}
	defer f.Close()

	log.Infof("Uploading file %s with Image ID %s", filePath, image.ID)
	imagedata.Upload(client, image.ID, f)

	// Validate the uploaded image.  If it's anything other than 'active'
	// then there's been a problem
	validImage, _ := images.Get(client, image.ID).Extract()
	if validImage.Status != "active" {
		log.Fatalf("Error uploading image, status is %s", validImage.Status)
	} else {
		log.Infof("Image uploaded successfully!")
		fmt.Println(image.ID)
	}
}

func pushOpenstackCmd() *cobra.Command {
	var (
		imageName string
	)
	cmd := &cobra.Command{
		Use:   "openstack",
		Short: "push image to OpenStack Image store (Glance)",
		Long: `Push image to OpenStack Image store (Glance).
		First argument specifies the path to a disk file.
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// Check that the file both exists, and can be read
			checkFile(path)

			client, err := clientconfig.NewServiceClient("image", nil)
			if err != nil {
				log.Fatalf("Error connecting to your OpenStack cloud: %s", err)
			}

			createOpenStackImage(path, imageName, client)
			return nil
		},
	}

	cmd.Flags().StringVar(&imageName, "img-name", "", "A unique name for the image, if blank the filename will be used")

	return cmd
}
