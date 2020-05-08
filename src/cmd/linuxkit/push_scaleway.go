package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const defaultScalewayVolumeSize = 10 // GB

func pushScaleway(args []string) {
	flags := flag.NewFlagSet("scaleway", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push scaleway [options] path\n\n", invoked)
		fmt.Printf("'path' is the full path to an EFI ISO image. It will be copied to a new Scaleway instance in order to create a Scaeway image out of it.\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	nameFlag := flags.String("img-name", "", "Overrides the name used to identify the image name in Scaleway's images. Defaults to the base of 'path' with the '.iso' suffix removed")
	accessKeyFlag := flags.String("access-key", "", "Access Key to connect to Scaleway API")
	secretKeyFlag := flags.String("secret-key", "", "Secret Key to connect to Scaleway API")
	sshKeyFlag := flags.String("ssh-key", os.Getenv("HOME")+"/.ssh/id_rsa", "SSH key file")
	instanceIDFlag := flags.String("instance-id", "", "Instance ID of a running Scaleway instance, with a second volume.")
	deviceNameFlag := flags.String("device-name", "/dev/vdb", "Device name on which the image will be copied")
	volumeSizeFlag := flags.Int("volume-size", 0, "Size of the volume to use (in GB). Defaults to size of the ISO file rounded up to GB")
	zoneFlag := flags.String("zone", defaultScalewayZone, "Select Scaleway zone")
	organizationIDFlag := flags.String("organization-id", "", "Select Scaleway's organization ID")
	noCleanFlag := flags.Bool("no-clean", false, "Do not remove temporary instance and volumes")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]

	name := getStringValue(scalewayNameVar, *nameFlag, "")
	accessKey := getStringValue(accessKeyVar, *accessKeyFlag, "")
	secretKey := getStringValue(secretKeyVar, *secretKeyFlag, "")
	sshKeyFile := getStringValue(sshKeyVar, *sshKeyFlag, "")
	instanceID := getStringValue(instanceIDVar, *instanceIDFlag, "")
	deviceName := getStringValue(deviceNameVar, *deviceNameFlag, "")
	volumeSize := getIntValue(volumeSizeVar, *volumeSizeFlag, 0)
	zone := getStringValue(zoneVar, *zoneFlag, defaultScalewayZone)
	organizationID := getStringValue(organizationIDVar, *organizationIDFlag, "")

	const suffix = ".iso"
	if name == "" {
		name = strings.TrimSuffix(path, suffix)
		name = filepath.Base(name)
	}

	client, err := NewScalewayClient(accessKey, secretKey, zone, organizationID)
	if err != nil {
		log.Fatalf("Unable to connect to Scaleway: %v", err)
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
			log.Fatalf("Error creating a Scaleway instance: %v", err)
		}

		err = client.BootInstanceAndWait(instanceID)
		if err != nil {
			log.Fatalf("Error booting instance: %v", err)
		}
	}

	volumeID, err := client.GetSecondVolumeID(instanceID)
	if err != nil {
		log.Fatalf("Error retrieving second volume ID: %v", err)
	}

	err = client.CopyImageToInstance(instanceID, path, sshKeyFile)
	if err != nil {
		log.Fatalf("Error copying ISO file to Scaleway's instance: %v", err)
	}

	err = client.WriteImageToVolume(instanceID, deviceName)
	if err != nil {
		log.Fatalf("Error writing ISO file to additional volume: %v", err)
	}

	err = client.TerminateInstance(instanceID)
	if err != nil {
		log.Fatalf("Error terminating Scaleway's instance: %v", err)
	}

	err = client.CreateScalewayImage(instanceID, volumeID, name)
	if err != nil {
		log.Fatalf("Error creating Scaleway image: %v", err)
	}

	if !*noCleanFlag {
		err = client.DeleteInstanceAndVolumes(instanceID)
		if err != nil {
			log.Fatalf("Error deleting Scaleway instance and volumes: %v", err)
		}
	}
}
