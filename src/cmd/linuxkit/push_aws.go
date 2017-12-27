package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

const timeoutVar = "LINUXKIT_UPLOAD_TIMEOUT"

func pushAWS(args []string) {
	flags := flag.NewFlagSet("aws", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push aws [options] path\n\n", invoked)
		fmt.Printf("'path' specifies the full path of an AWS image. It will be uploaded to S3 and an AMI will be created from it.\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	timeoutFlag := flags.Int("timeout", 0, "Upload timeout in seconds")
	bucketFlag := flags.String("bucket", "", "S3 Bucket to upload to. *Required*")
	nameFlag := flags.String("img-name", "", "Overrides the name used to identify the file in Amazon S3 and the VM image. Defaults to the base of 'path' with the file extension removed.")
	enaFlag := flags.Bool("ena", false, "Enable ENA networking")
	sriovNetFlag := flags.String("sriov", "", "SRIOV network support, set to 'simple' to enable 82599 VF networking")

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

	timeout := getIntValue(timeoutVar, *timeoutFlag, 600)
	bucket := getStringValue(bucketVar, *bucketFlag, "")
	name := getStringValue(nameVar, *nameFlag, "")
	if *sriovNetFlag == "" {
		sriovNetFlag = nil
	}

	sess := session.Must(session.NewSession())
	storage := s3.New(sess)

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelFn()

	if bucket == "" {
		log.Fatalf("Please provide the bucket to use")
	}

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	if name == "" {
		name = strings.TrimSuffix(path, filepath.Ext(path))
		name = filepath.Base(name)
	}

	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("Error reading file information: %v", err)
	}

	dst := name + filepath.Ext(path)
	putParams := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(dst),
		Body:          f,
		ContentLength: aws.Int64(fi.Size()),
		ContentType:   aws.String("application/octet-stream"),
	}
	log.Debugf("PutObject:\n%v", putParams)

	_, err = storage.PutObjectWithContext(ctx, putParams)
	if err != nil {
		log.Fatalf("Error uploading to S3: %v", err)
	}

	compute := ec2.New(sess)

	importParams := &ec2.ImportSnapshotInput{
		Description: aws.String(fmt.Sprintf("LinuxKit: %s", name)),
		DiskContainer: &ec2.SnapshotDiskContainer{
			Description: aws.String(fmt.Sprintf("LinuxKit: %s disk", name)),
			Format:      aws.String("raw"),
			UserBucket: &ec2.UserBucket{
				S3Bucket: aws.String(bucket),
				S3Key:    aws.String(dst),
			},
		},
	}
	log.Debugf("ImportSnapshot:\n%v", importParams)

	resp, err := compute.ImportSnapshot(importParams)
	if err != nil {
		log.Fatalf("Error importing snapshot: %v", err)
	}

	var snapshotID *string
	for {
		describeParams := &ec2.DescribeImportSnapshotTasksInput{
			ImportTaskIds: []*string{
				resp.ImportTaskId,
			},
		}
		log.Debugf("DescribeImportSnapshotTask:\n%v", describeParams)
		status, err := compute.DescribeImportSnapshotTasks(describeParams)
		if err != nil {
			log.Fatalf("Error getting import snapshot status: %v", err)
		}
		if len(status.ImportSnapshotTasks) == 0 {
			log.Fatalf("Unable to get import snapshot task status")
		}
		if *status.ImportSnapshotTasks[0].SnapshotTaskDetail.Status != "completed" {
			progress := "0"
			if status.ImportSnapshotTasks[0].SnapshotTaskDetail.Progress != nil {
				progress = *status.ImportSnapshotTasks[0].SnapshotTaskDetail.Progress
			}
			log.Debugf("Task %s is %s%% complete. Waiting 60 seconds...\n", *resp.ImportTaskId, progress)
			time.Sleep(60 * time.Second)
			continue
		}
		snapshotID = status.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId
		break
	}

	if snapshotID == nil {
		log.Fatalf("SnapshotID unavailable after import completed")
	} else {
		log.Debugf("SnapshotID: %s", snapshotID)
	}

	regParams := &ec2.RegisterImageInput{
		Name:         aws.String(name), // Required
		Architecture: aws.String("x86_64"),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          snapshotID,
					VolumeType:          aws.String("standard"),
				},
			},
		},
		Description:        aws.String(fmt.Sprintf("LinuxKit: %s image", name)),
		RootDeviceName:     aws.String("/dev/sda1"),
		VirtualizationType: aws.String("hvm"),
		EnaSupport:         enaFlag,
		SriovNetSupport:    sriovNetFlag,
	}
	log.Debugf("RegisterImage:\n%v", regParams)
	regResp, err := compute.RegisterImage(regParams)
	if err != nil {
		log.Fatalf("Error registering the image: %s; %v", name, err)
	}
	log.Infof("Created AMI: %s", *regResp.ImageId)
}
