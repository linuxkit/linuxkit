package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
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
	authurl := flags.String("authurl", "", "The URL of the OpenStack identity service, i.e https://keystone.example.com:5000/v3")
	usernameFlag := flags.String("username", "", "Username with permissions to upload image")
	passwordFlag := flags.String("password", "", "Password for the Username")
	projectName := flags.String("project", "", "Name of the Project (aka Tenant) to be used")
	userDomainFlag := flags.String("domain", "Default", "Domain name")

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

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: *authurl,
		Username:         *usernameFlag,
		Password:         *passwordFlag,
		DomainName:       *userDomainFlag,
		TenantName:       *projectName,
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatalf("Failed to authenticate")
	}

	createOpenStackImage(filePath, *imageName, provider)
}

func createOpenStackImage(filePath string, imageName string, provider *gophercloud.ProviderClient) {
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

	client, err := openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Fatalf("Unable to create Image V2 client")
	}
	imageOpts := images.CreateOpts{
		Name:            imageName,
		ContainerFormat: "bare",
		DiskFormat:      fileExtension,
	}
	image, err := images.Create(client, imageOpts).Extract()
	if err != nil {
		log.Fatalf("Error creating image")
	}

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
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
	}
}
