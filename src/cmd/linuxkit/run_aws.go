package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

const (
	defaultAWSMachine  = "t2.micro"
	defaultAWSDiskSize = 0
	defaultAWSDiskType = "gp2"
	defaultAWSZone     = "a"
	// Environment variables. Some are non-standard
	awsMachineVar  = "AWS_MACHINE"   // non-standard
	awsDiskSizeVar = "AWS_DISK_SIZE" // non-standard
	awsDiskTypeVar = "AWS_DISK_TYPE" // non-standard
	awsZoneVar     = "AWS_ZONE"      // non-standard
)

// Process the run arguments and execute run
func runAWS(args []string) {
	flags := flag.NewFlagSet("aws", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run aws [options] [name]\n\n", invoked)
		fmt.Printf("'name' is the name of an AWS image that has already been\n")
		fmt.Printf(" uploaded using 'linuxkit push'\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	machineFlag := flags.String("machine", defaultAWSMachine, "AWS Machine Type")
	diskSizeFlag := flags.Int("disk-size", 0, "Size of system disk in GB")
	diskTypeFlag := flags.String("disk-type", defaultAWSDiskType, "AWS Disk Type")
	zoneFlag := flags.String("zone", defaultAWSZone, "AWS Availability Zone")
	sgFlag := flags.String("security-group", "", "Security Group ID")

	data := flags.String("data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	dataPath := flags.String("data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

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

	if *data != "" && *dataPath != "" {
		log.Fatal("Cannot specify both -data and -data-file")
	}

	if *dataPath != "" {
		dataB, err := ioutil.ReadFile(*dataPath)
		if err != nil {
			log.Fatalf("Unable to read metadata file: %v", err)
		}
		*data = string(dataB)
	}
	// data must be base64 encoded
	*data = base64.StdEncoding.EncodeToString([]byte(*data))

	machine := getStringValue(awsMachineVar, *machineFlag, defaultAWSMachine)
	diskSize := getIntValue(awsDiskSizeVar, *diskSizeFlag, defaultAWSDiskSize)
	diskType := getStringValue(awsDiskTypeVar, *diskTypeFlag, defaultAWSDiskType)
	zone := os.Getenv("AWS_REGION") + getStringValue(awsZoneVar, *zoneFlag, defaultAWSZone)

	sess := session.Must(session.NewSession())
	compute := ec2.New(sess)

	// 1. Find AMI
	filter := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String(name)},
			},
		},
	}
	results, err := compute.DescribeImages(filter)
	if err != nil {
		log.Fatalf("Unable to describe images: %s", err)
	}
	if len(results.Images) < 1 {
		log.Fatalf("Unable to find image with name %s", name)
	}
	if len(results.Images) > 1 {
		log.Warnf("Found multiple images with the same name, using the first one")
	}
	imageID := results.Images[0].ImageId

	// 2. Create Instance
	params := &ec2.RunInstancesInput{
		ImageId:      imageID,
		InstanceType: aws.String(machine),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String(zone),
		},
		SecurityGroupIds: []*string{sgFlag},
		UserData:         data,
	}
	runResult, err := compute.RunInstances(params)
	if err != nil {
		log.Fatalf("Unable to run instance: %s", err)

	}
	instanceID := runResult.Instances[0].InstanceId
	log.Infof("Created instance %s", *instanceID)

	instanceFilter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []*string{instanceID},
			},
		},
	}

	if err = compute.WaitUntilInstanceRunning(instanceFilter); err != nil {
		log.Fatalf("Error waiting for instance to start: %s", err)
	}
	log.Infof("Instance %s is running", *instanceID)

	if diskSize > 0 {
		// 3. Create EBS Volume
		diskParams := &ec2.CreateVolumeInput{
			AvailabilityZone: aws.String(zone),
			Size:             aws.Int64(int64(diskSize)),
			VolumeType:       aws.String(diskType),
		}
		log.Debugf("CreateVolume:\n%v\n", diskParams)

		volume, err := compute.CreateVolume(diskParams)
		if err != nil {
			log.Fatalf("Error creating volume: %s", err)
		}

		waitVol := &ec2.DescribeVolumesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("volume-id"),
					Values: []*string{volume.VolumeId},
				},
			},
		}

		log.Infof("Waiting for volume %s to be available", *volume.VolumeId)

		if err := compute.WaitUntilVolumeAvailable(waitVol); err != nil {
			log.Fatalf("Error waiting for volume to be available: %s", err)
		}

		log.Infof("Attaching volume %s to instance %s", *volume.VolumeId, *instanceID)
		volParams := &ec2.AttachVolumeInput{
			Device:     aws.String("/dev/sda2"),
			InstanceId: instanceID,
			VolumeId:   volume.VolumeId,
		}
		_, err = compute.AttachVolume(volParams)
		if err != nil {
			log.Fatalf("Error attaching volume to instance: %s", err)
		}
	}

	log.Warnf("AWS doesn't stream serial console output.\n Please use the AWS Management Console to obtain this output \n Console output will be displayed when the instance has been stopped.")
	log.Warn("Waiting for instance to stop...")

	if err = compute.WaitUntilInstanceStopped(instanceFilter); err != nil {
		log.Fatalf("Error waiting for instance to stop: %s", err)
	}

	consoleParams := &ec2.GetConsoleOutputInput{
		InstanceId: instanceID,
	}
	output, err := compute.GetConsoleOutput(consoleParams)
	if err != nil {
		log.Fatalf("Error getting output from instance %s: %s", *instanceID, err)
	}

	if output.Output == nil {
		log.Warn("No Console Output found")
	} else {
		out, err := base64.StdEncoding.DecodeString(*output.Output)
		if err != nil {
			log.Fatalf("Error decoding output: %s", err)
		}
		fmt.Printf(string(out) + "\n")
	}
	log.Infof("Terminating instance %s", *instanceID)
	terminateParams := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{instanceID},
	}
	if _, err := compute.TerminateInstances(terminateParams); err != nil {
		log.Fatalf("Error terminating instance %s", *instanceID)
	}
	if err = compute.WaitUntilInstanceTerminated(instanceFilter); err != nil {
		log.Fatalf("Error waiting for instance to terminate: %s", err)
	}
}
