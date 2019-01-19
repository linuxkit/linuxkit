package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/utils/openstack/clientconfig"

	log "github.com/sirupsen/logrus"
)

// Process the run arguments and execute run
func pushOpenstack(args []string) {
	flags := flag.NewFlagSet("openstack", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push openstack [options] path\n\n", invoked)
		fmt.Printf("'path' is the full path to an image that will be uploaded to an OpenStack Image store (Glance)\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	imageName := flags.String("img-name", "", "A unique name for the image, if blank the filename will be used")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	filePath := remArgs[0]
	// Check that the file both exists, and can be read
	checkFile(filePath)

	client, err := clientconfig.NewServiceClient("image", nil)
	if err != nil {
		log.Fatalf("Error connecting to your OpenStack cloud: %s", err)
	}

	createOpenStackImage(filePath, *imageName, client)
}

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

	if supportedExtension == false {
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
