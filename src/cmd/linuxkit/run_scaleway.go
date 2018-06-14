package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	defaultScalewayInstanceType = "VC1S"
	defaultScalewayRegion       = "par1"

	scalewayNameVar = "SCW_IMAGE_NAME"   // non-standard
	tokenVar        = "SCW_TOKEN"        // non-standard
	sshKeyVar       = "SCW_SSH_KEY_FILE" // non-standard
	instanceIDVar   = "SCW_INSTANCE_ID"  // non-standard
	deviceNameVar   = "SCW_DEVICE_NAME"  // non-standard
	regionVar       = "SCW_TARGET_REGION"

	instanceTypeVar = "SCW_RUN_TYPE" // non-standard
)

func runScaleway(args []string) {
	flags := flag.NewFlagSet("scaleway", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run scaleway [options] [name]\n\n", invoked)
		fmt.Printf("'name' is the name of a Scaleway image that has alread \n")
		fmt.Printf("been uploaded using 'linuxkit push'\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	instanceTypeFlag := flags.String("instance-type", defaultScalewayInstanceType, "Scaleway instance type")
	instanceNameFlag := flags.String("instance-name", "linuxkit", "Name of the create instance, default to the image name")
	tokenFlag := flags.String("token", "", "Token to connect to Scaleway API")
	regionFlag := flags.String("region", defaultScalewayRegion, "Select Scaleway region")
	cleanFlag := flags.Bool("clean", false, "Remove instance")
	noAttachFlag := flags.Bool("no-attach", false, "Don't attach to serial port, you will have to connect to instance manually")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the name of the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	name := remArgs[0]

	instanceType := getStringValue(instanceTypeVar, *instanceTypeFlag, defaultScalewayInstanceType)
	instanceName := getStringValue("", *instanceNameFlag, name)
	token := getStringValue(tokenVar, *tokenFlag, "")
	region := getStringValue(regionVar, *regionFlag, defaultScalewayRegion)

	client, err := NewScalewayClient(token, region)
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

	if !*noAttachFlag {
		err = client.ConnectSerialPort(instanceID)
		if err != nil {
			log.Fatalf("Unable to connect to serial port: %v", err)
		}
	}

	if *cleanFlag {
		err = client.TerminateInstance(instanceID)
		if err != nil {
			log.Fatalf("Unable to stop instance: %v", err)
		}

		err = client.DeleteInstanceAndVolumes(instanceID)
		if err != nil {
			log.Fatalf("Unable to delete instance: %v", err)
		}
	}

}
