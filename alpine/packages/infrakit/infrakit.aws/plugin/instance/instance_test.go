package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	mock_ec2 "github.com/docker/infrakit.aws/mock/ec2"
	"github.com/docker/infrakit/spi/instance"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	tags = map[string]string{"group": "workers"}
)

func TestInstanceLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clientMock := mock_ec2.NewMockEC2API(ctrl)
	provisioner := Provisioner{Client: clientMock}

	// Create an instance.

	instanceID := "test-id"

	clientMock.EXPECT().RunInstances(gomock.Any()).
		Return(&ec2.Reservation{Instances: []*ec2.Instance{{InstanceId: &instanceID}}}, nil)

	tagRequest := ec2.CreateTagsInput{
		Resources: []*string{&instanceID},
		Tags: []*ec2.Tag{
			{Key: aws.String("group"), Value: aws.String("workers")},
			{Key: aws.String("test"), Value: aws.String("aws-create-test")},
		},
	}
	clientMock.EXPECT().CreateTags(&tagRequest).Return(&ec2.CreateTagsOutput{}, nil)

	// TODO(wfarner): Test user-data and private IP plumbing.
	id, err := provisioner.Provision(instance.Spec{Properties: &inputJSON, Tags: tags})

	require.NoError(t, err)
	require.Equal(t, instanceID, string(*id))

	// Destroy the instance.

	clientMock.EXPECT().TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{&instanceID}}).
		Return(&ec2.TerminateInstancesOutput{
			TerminatingInstances: []*ec2.InstanceStateChange{{InstanceId: &instanceID}}},
			nil)

	require.NoError(t, provisioner.Destroy(instance.ID(instanceID)))
}

func TestCreateInstanceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	clientMock := mock_ec2.NewMockEC2API(ctrl)

	runError := errors.New("request failed")
	clientMock.EXPECT().RunInstances(gomock.Any()).Return(&ec2.Reservation{}, runError)

	provisioner := NewInstancePlugin(clientMock)
	properties := json.RawMessage("{}")
	id, err := provisioner.Provision(instance.Spec{Properties: &properties, Tags: tags})

	require.Error(t, err)
	require.Nil(t, id)
}

func TestDestroyInstanceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	clientMock := mock_ec2.NewMockEC2API(ctrl)

	instanceID := "test-id"

	runError := errors.New("request failed")
	clientMock.EXPECT().TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: []*string{&instanceID}}).
		Return(nil, runError)

	provisioner := NewInstancePlugin(clientMock)
	require.Error(t, provisioner.Destroy(instance.ID(instanceID)))
}

func describeInstancesResponse(
	instanceIds [][]string,
	tags map[string]string,
	nextToken *string) *ec2.DescribeInstancesOutput {

	ec2Tags := []*ec2.Tag{}
	for key, value := range tags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	reservations := []*ec2.Reservation{}
	for _, ids := range instanceIds {
		instances := []*ec2.Instance{}
		for _, id := range ids {
			instances = append(instances, &ec2.Instance{
				InstanceId:       aws.String(id),
				PrivateIpAddress: aws.String("127.0.0.1"),
				Tags:             ec2Tags,
			})
		}
		reservations = append(reservations, &ec2.Reservation{Instances: instances})
	}

	return &ec2.DescribeInstancesOutput{NextToken: nextToken, Reservations: reservations}
}

func TestDescribeInstancesRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var nextToken *string
	request := describeGroupRequest(tags, nextToken)

	require.Equal(t, nextToken, request.NextToken)

	requireFilter := func(name, value string) {
		for _, filter := range request.Filters {
			if *filter.Name == name {
				for _, filterValue := range filter.Values {
					if *filterValue == value {
						// Match found
						return
					}
				}
			}
		}
		require.Fail(t, fmt.Sprintf("Did not have filter %s/%s", name, value))
	}
	for key, value := range tags {
		requireFilter(fmt.Sprintf("tag:%s", key), value)
	}

	nextToken = aws.String("page-2")
	request = describeGroupRequest(tags, nextToken)
	require.Equal(t, nextToken, request.NextToken)
}

func TestListGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	clientMock := mock_ec2.NewMockEC2API(ctrl)

	page2Token := "page2"

	// Split instance IDs across multiple reservations and request pages.
	gomock.InOrder(
		clientMock.EXPECT().DescribeInstances(describeGroupRequest(tags, nil)).
			Return(describeInstancesResponse([][]string{
				{"a", "b", "c"},
				{"d", "e"},
			}, tags, &page2Token), nil),
		clientMock.EXPECT().DescribeInstances(describeGroupRequest(tags, &page2Token)).
			Return(describeInstancesResponse([][]string{{"f", "g"}}, tags, nil), nil),
	)

	provisioner := NewInstancePlugin(clientMock)
	descriptions, err := provisioner.DescribeInstances(tags)

	require.NoError(t, err)
	id := instance.LogicalID("127.0.0.1")
	require.Equal(t, []instance.Description{
		{ID: "a", LogicalID: &id, Tags: tags},
		{ID: "b", LogicalID: &id, Tags: tags},
		{ID: "c", LogicalID: &id, Tags: tags},
		{ID: "d", LogicalID: &id, Tags: tags},
		{ID: "e", LogicalID: &id, Tags: tags},
		{ID: "f", LogicalID: &id, Tags: tags},
		{ID: "g", LogicalID: &id, Tags: tags},
	}, descriptions)
}

var inputJSON = json.RawMessage(`{
    "tags": {"test": "aws-create-test"},
    "run_instances_input": {
        "BlockDeviceMappings": [
          {
            "DeviceName": "/dev/sdb",
            "Ebs": {
                "DeleteOnTermination": true,
                "VolumeSize": 64,
                "VolumeType": "gp2"
            }
          }
        ],
        "EbsOptimized": false,
        "ImageId": "ami-30ee0d50",
        "InstanceType": "t2.micro",
        "Monitoring": {
            "Enabled": true
        },
        "NetworkInterfaces": [
          {
            "AssociatePublicIpAddress": true,
            "DeleteOnTermination": true,
            "DeviceIndex": 0,
            "Groups": [
                "sg-973491f0"
            ],
            "SubnetId": "subnet-2"
          }
        ],
        "Placement": {
            "AvailabilityZone": "us-west-2a"
        },
        "UserData": "A string; which must && be base64 encoded"
    }
}
`)
