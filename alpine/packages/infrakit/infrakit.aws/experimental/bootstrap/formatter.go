package bootstrap

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strings"
	"text/template"
	"time"
)

func formatVolumes(config client.ConfigProvider, swim fakeSWIMSchema, volumeIDs []*string) error {
	log.Info("Formatting Swarm data EBS volumes")

	// TODO(wfarner): Could we instead format on the bootstrap volume, as part of other bootstrap operations
	// (like `docker swarm init`)?

	// On the host OS we are using to format, mounted block devices appear as '/dev/xvdX' even though we ask EC2
	// to mount as '/dev/sdX'.  This behavior is explicitly mentioned in
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html#device-name-limits
	volumeLetters := []string{"f", "g", "h", "i", "j", "k", "l"}
	volumeToName := map[string]string{}
	hostDeviceNames := []string{}
	for index, volumeID := range volumeIDs {
		letter := volumeLetters[index]
		volumeToName[*volumeID] = fmt.Sprintf("/dev/sd%s", letter)
		hostDeviceNames = append(hostDeviceNames, fmt.Sprintf("/dev/xvd%s", letter))
	}

	userDataTemplate := template.Must(template.New("userdata").Parse(userData))
	buffer := bytes.Buffer{}
	err := userDataTemplate.Execute(&buffer, map[string]string{"Devices": strings.Join(hostDeviceNames, " ")})
	if err != nil {
		return fmt.Errorf("Failed to generate UserData: %s", err)
	}

	// TODO(wfarner): Need to pick the appropriate image based on the region.
	ec2Client := ec2.New(config)
	instance, err := ec2Client.RunInstances(&ec2.RunInstancesInput{
		InstanceType: aws.String("t2.micro"),
		KeyName:      swim.managers().Config.RunInstancesInput.KeyName,
		ImageId:      aws.String("ami-2ef48339"),
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String(swim.availabilityZone()),
		},
		UserData: aws.String(base64.StdEncoding.EncodeToString([]byte(buffer.String()))),
		MinCount: aws.Int64(1),
		MaxCount: aws.Int64(1),
	})
	if err != nil {
		return fmt.Errorf("Failed to start formatter instance: %s", err)
	}
	formatterInstance := instance.Instances[0].InstanceId

	_, err = ec2Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{formatterInstance},
		Tags: []*ec2.Tag{
			swim.cluster().resourceTag(),
			{
				Key:   aws.String("Name"),
				Value: aws.String(fmt.Sprintf("%s formatter", swim.cluster().name)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error while tagging formatter instance: %s", err)
	}

	log.Infof("  created EBS formatter instance %s, waiting for it to run", *formatterInstance)
	getFormatter := ec2.DescribeInstancesInput{InstanceIds: []*string{formatterInstance}}
	err = ec2Client.WaitUntilInstanceRunning(&getFormatter)
	if err != nil {
		return fmt.Errorf("Error while waiting for formatter instance to start: %s", err)
	}

	for volumeID, deviceName := range volumeToName {
		_, err = ec2Client.AttachVolume(&ec2.AttachVolumeInput{
			InstanceId: formatterInstance,
			VolumeId:   aws.String(volumeID),
			Device:     aws.String(deviceName),
		})
		if err != nil {
			return fmt.Errorf("Error while attaching volume %s for formatting: %s", volumeID, err)
		}
	}

	// TODO(wfarner): Include a timeout for this.
	log.Info("  attached volumes to formatter, waiting for it to boot and format")
	formattingFailed := true
	for {
		time.Sleep(10 * time.Second)

		// This is an especially primitive way to collect exit status, but the upside is that it requires
		// no additional machinery or network access.
		console, err := ec2Client.GetConsoleOutput(&ec2.GetConsoleOutputInput{InstanceId: formatterInstance})
		if err != nil {
			return fmt.Errorf("Error while fetching formatter instance console: %s", err)
		}

		if console.Output != nil {
			consoleData, err := base64.StdEncoding.DecodeString(*console.Output)
			if err != nil {
				return fmt.Errorf("Error while decoding formatter console text: %s", err)
			}

			consoleText := string(consoleData)
			if strings.Contains(consoleText, "INFRAKIT FORMATTING FINISHED: success") {
				log.Info("  formatting complete")
				formattingFailed = false
				break
			} else if strings.Contains(consoleText, "INFRAKIT FORMATTING FINISHED: unexpected exit") {
				log.Fatal("  failed to format EBS volumes.  See console output below.")
				log.Fatal(consoleText)
				break
			}
		}
	}

	_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{formatterInstance}})
	if err != nil {
		return fmt.Errorf("Error while terminating formatter instance: %s", err)
	}

	log.Info("  waiting for formatter instance to terminate")
	err = ec2Client.WaitUntilInstanceTerminated(&getFormatter)
	if err != nil {
		return fmt.Errorf("Error while waiting for formatter instance to terminate: %s", err)
	}

	if formattingFailed {
		return errors.New("Failed to format EBS voluems")
	}
	return nil
}

const userData = `#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

ls -l /dev

completed=false
onexit () {
  if [ "$completed" = true ]
  then
    echo 'INFRAKIT FORMATTING FINISHED: success'
  else
    echo 'INFRAKIT FORMATTING FINISHED: unexpected exit'
  fi
}

trap onexit EXIT

for device in {{.Devices}}
do
  mkfs -t ext4 $device
done

completed=true
`
