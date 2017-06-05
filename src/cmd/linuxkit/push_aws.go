package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

const timeoutVar = "LINUXKIT_UPLOAD_TIMEOUT"

func pushAWS(args []string) {
	awsCmd := flag.NewFlagSet("aws", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	awsCmd.Usage = func() {
		fmt.Printf("USAGE: %s push aws [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies the full path of an image file which will be uploaded\n")
		fmt.Printf("Options:\n\n")
		awsCmd.PrintDefaults()
	}
	timeoutFlag := awsCmd.Int("timeout", 0, "Upload timeout in seconds")
	bucketFlag := awsCmd.String("bucket", "", "S3 Bucket to upload to. *Required*")
	nameFlag := awsCmd.String("img-name", "", "Overrides the Name used to identify the file in Amazon S3 and Image. Defaults to [name] with the file extension removed.")

	if err := awsCmd.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := awsCmd.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		awsCmd.Usage()
		os.Exit(1)
	}
	src := remArgs[0]

	timeout := getIntValue(timeoutVar, *timeoutFlag, 600)
	bucket := getStringValue(bucketVar, *bucketFlag, "")
	name := getStringValue(nameVar, *nameFlag, "")

	sess := session.Must(session.NewSession())
	storage := s3.New(sess)

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelFn()

	if bucket == "" {
		log.Fatalf("No bucket specified. Please provide one using the -bucket flag")
	}

	f, err := os.Open(src)
	if err != nil {
		log.Fatalf("Error opening file: %s", err)
	}
	defer f.Close()

	if name == "" {
		name = strings.TrimSuffix(name, filepath.Ext(src))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("error reading file: %s", err)
	}

	dst := name + filepath.Ext(src)
	putParams := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(dst),
		Body:          f,
		ContentLength: aws.Int64(int64(len(content))),
		ContentType:   aws.String("application/octet-stream"),
	}
	log.Debugf("PutObject:\n%v", putParams)

	_, err = storage.PutObjectWithContext(ctx, putParams)
	if err != nil {
		log.Fatalf("Error uploading to S3: %s", err)
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
		log.Fatalf("Error importing snapshot: %s", err)
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
			log.Fatalf("Error getting import snapshot status: %s", err)
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
	}
	log.Debugf("RegisterImage:\n%v", regParams)
	regResp, err := compute.RegisterImage(regParams)
	if err != nil {
		log.Fatalf("Error registering the image: %s", err)
	}
	log.Infof("Created AMI: %s", *regResp.ImageId)
}
