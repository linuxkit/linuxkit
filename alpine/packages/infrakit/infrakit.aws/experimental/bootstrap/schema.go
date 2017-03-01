package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/docker/infrakit.aws/plugin/instance"
	"github.com/docker/infrakit/spi/group"
	"strings"
)

const (
	workerType  = "worker"
	managerType = "manager"
	s3File      = "config.swim"
)

type clusterID struct {
	region string
	name   string
}

func (c clusterID) getAWSClient() client.ConfigProvider {
	providers := []credentials.Provider{
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(session.New())},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{},
	}

	return session.New(aws.NewConfig().
		WithRegion(c.region).
		WithCredentialsChainVerboseErrors(true).
		WithCredentials(credentials.NewChainCredentials(providers)).
		WithLogger(&logger{}))
}

func (c clusterID) url() string {
	return fmt.Sprintf(
		"https://infrakit-cluster.s3-%s.amazonaws.com/%s/%s",
		c.region,
		c.name,
		s3File)
}

func (c clusterID) resourceFilter(vpcID string) []*ec2.Filter {
	return []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcID)},
		},
		c.clusterFilter(),
	}
}

func (c clusterID) clusterFilter() *ec2.Filter {
	return &ec2.Filter{
		Name:   aws.String("tag:infrakit-cluster"),
		Values: []*string{aws.String(c.name)},
	}
}

func (c clusterID) roleName() string {
	return fmt.Sprintf("%s-ManagerRole", c.name)
}

func (c clusterID) managerPolicyName() string {
	return fmt.Sprintf("%s-ManagerPolicy", c.name)
}

func (c clusterID) instanceProfileName() string {
	return fmt.Sprintf("%s-ManagerProfile", c.name)
}

func (c clusterID) resourceTag() *ec2.Tag {
	return &ec2.Tag{
		Key:   aws.String("infrakit-cluster"),
		Value: aws.String(c.name),
	}
}

type instanceGroup struct {
	Name   group.ID
	Type   string
	Size   int
	Config instance.CreateInstanceRequest
}

func (i instanceGroup) isManager() bool {
	return i.Type == managerType
}

type fakeSWIMSchema struct {
	Driver      string
	ClusterName string
	ManagerIPs  []string
	Groups      []instanceGroup
}

func (s *fakeSWIMSchema) cluster() clusterID {
	az := s.availabilityZone()
	return clusterID{region: az[:len(az)-1], name: s.ClusterName}
}

func (s *fakeSWIMSchema) push() error {
	swimData, err := json.Marshal(*s)
	if err != nil {
		return err
	}

	s3Client := s3.New(s.cluster().getAWSClient())

	bucket := aws.String("infrakit-cluster")
	head := &s3.HeadBucketInput{Bucket: bucket}

	_, err = s3Client.HeadBucket(head)
	if err != nil {
		// The bucket does not appear to exist.  Try to create it.
		bucketCreateResult, err := s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: bucket,
		})
		if err != nil {
			return err
		}

		log.Infof("Created S3 bucket: %s", *bucketCreateResult.Location)

		err = s3Client.WaitUntilBucketExists(head)
		if err != nil {
			return err
		}
	}

	key := aws.String(fmt.Sprintf("%s/%s", s.ClusterName, s3File))

	// Note - this will overwrite an existing object.
	putRequest, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		// TODO(wfarner): Explore tightening permissions, as these URLs are reasonably guessable and could
		// potentially contain sensitive information in the future.
		ACL:         aws.String("public-read"),
		Body:        bytes.NewReader(swimData),
		ContentType: aws.String("application/json"),
	})

	err = putRequest.Send()
	if err != nil {
		return err
	}

	if putRequest.HTTPRequest.URL.String() != s.cluster().url() {
		log.Warnf(
			"Expected config URL %s, but received %s",
			s.cluster().url(),
			putRequest.HTTPRequest.URL.String())
	}

	return nil
}

func (s *fakeSWIMSchema) managers() instanceGroup {
	for _, group := range s.Groups {
		if group.isManager() {
			return group
		}
	}
	panic("No manager group found")
}

func (s *fakeSWIMSchema) mutateManagers(op func(*instanceGroup)) {
	s.mutateGroups(func(group *instanceGroup) {
		if group.isManager() {
			op(group)
		}
	})
}

func (s *fakeSWIMSchema) mutateGroups(op func(*instanceGroup)) {
	for i, group := range s.Groups {
		op(&group)
		s.Groups[i] = group
	}
}

func applyInstanceDefaults(r *ec2.RunInstancesInput) {
	if r.InstanceType == nil {
		r.InstanceType = aws.String("t2.micro")
	}

	if r.NetworkInterfaces == nil || len(r.NetworkInterfaces) == 0 {
		r.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				DeviceIndex:              aws.Int64(0),
			},
		}
	}
}

func (s *fakeSWIMSchema) applyDefaults() {
	s.mutateGroups(func(group *instanceGroup) {
		if group.Type == managerType {
			bootLeaderLastOctet := 4
			s.ManagerIPs = []string{}
			for i := 0; i < group.Size; i++ {
				s.ManagerIPs = append(s.ManagerIPs, fmt.Sprintf("192.168.33.%d", bootLeaderLastOctet+i))
			}
		}

		applyInstanceDefaults(&group.Config.RunInstancesInput)
	})
}

func (s *fakeSWIMSchema) validate() error {
	errs := []string{}

	addError := func(format string, a ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, a...))
	}

	managerGroups := 0
	workerGroups := 0
	for _, group := range s.Groups {
		switch group.Type {
		case managerType:
			managerGroups++
		case workerType:
			workerGroups++
		default:
			errs = append(
				errs,
				fmt.Sprintf(
					"Invalid instance type '%s', must be %s or %s",
					group.Type,
					workerType,
					managerType))
		}
	}

	if managerGroups != 1 {
		addError("Must specify exactly one group of type %s", managerType)
	}

	/*
		if workerGroups == 0 {
			addError("Must specify exactly one group of type %s", workerType)
		}
	*/

	if s.ClusterName == "" {
		addError("Must specify ClusterName")
	}

	for _, group := range s.Groups {
		if group.isManager() {
			if group.Size != 1 && group.Size != 3 && group.Size != 5 {
				addError("Group %s Size must be 1, 3, or 5", group.Name)
			}
		} else {
			if group.Size < 1 {
				addError("Group %s Size must be at least 1", group.Name)
			}
		}
	}

	validateGroup := func(gid group.ID, group instanceGroup) {
		errorPrefix := fmt.Sprintf("In group %s: ", gid)

		if group.Config.RunInstancesInput.Placement == nil {
			addError(errorPrefix + "run_instance_input.Placement must be set")
		} else if group.Config.RunInstancesInput.Placement.AvailabilityZone == nil ||
			*group.Config.RunInstancesInput.Placement.AvailabilityZone == "" {

			addError(errorPrefix + "run_instance_nput.Placement.AvailabilityZone must be set")
		}
	}

	// MVP restriction - all groups must be in the same Availability Zone.
	firstAz := ""
	for _, group := range s.Groups {
		validateGroup(group.Name, group)

		if group.Config.RunInstancesInput.Placement != nil {
			az := *group.Config.RunInstancesInput.Placement.AvailabilityZone
			if firstAz == "" {
				firstAz = az
			} else if az != firstAz {
				addError(
					"All groups must specify the same run_instance_nput.Placement.AvailabilityZone")
				break
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (s *fakeSWIMSchema) availabilityZone() string {
	for _, group := range s.Groups {
		return *group.Config.RunInstancesInput.Placement.AvailabilityZone
	}
	panic("No groups")
}
