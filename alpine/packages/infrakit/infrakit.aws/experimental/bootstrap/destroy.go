package bootstrap

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

func destroyInstances(config client.ConfigProvider, cluster clusterID, vpcID string) {
	log.Info("Destroying instances")
	ec2Client := ec2.New(config)

	instancesResp, err := ec2Client.DescribeInstances(
		&ec2.DescribeInstancesInput{Filters: cluster.resourceFilter(vpcID)})
	if err != nil {
		log.Errorf("  failed to fetch instances: %s", err)
		return
	}

	instanceIDs := []*string{}
	for _, reservation := range instancesResp.Reservations {
		for _, instance := range reservation.Instances {
			instanceIDs = append(instanceIDs, instance.InstanceId)
		}
	}

	if len(instanceIDs) > 0 {
		nonPointerIds := []string{}
		for _, id := range instanceIDs {
			nonPointerIds = append(nonPointerIds, *id)
			log.Infof("  %s", *id)
		}

		_, err = ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIds: instanceIDs})
		if err != nil {
			log.Errorf("  failed to terminate instances: %s", err)
			return
		}

		// TODO(wfarner): Need a more robust routine here since instances can self-replicate.  For example,
		// the describe/terminate sequence here could race with a manager node trying to reach a target
		// instance count.
		err = ec2Client.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{InstanceIds: instanceIDs})
		if err != nil {
			log.Warnf("  error while waiting for instances to terminate: %s", err)
		}
	} else {
		log.Warnf("  did not find any instances to terminate")
	}
}

func destroyEBSVolues(config client.ConfigProvider, cluster clusterID) {
	log.Info("Destroying EBS volumes")
	ec2Client := ec2.New(config)

	volumes, err := ec2Client.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{cluster.clusterFilter()},
	})
	if err != nil {
		log.Warnf("  error while describing volumes: %s", err)
		return
	}

	for _, volume := range volumes.Volumes {
		_, err = ec2Client.DeleteVolume(&ec2.DeleteVolumeInput{VolumeId: volume.VolumeId})
		if err == nil {
			log.Infof("  %s", *volume.VolumeId)
		} else {
			log.Warnf("  error while deleting volume %s: %s", *volume.VolumeId, err)
		}
	}
}

func destroyAccessRoles(config client.ConfigProvider, cluster clusterID) {
	log.Info("Destroying IAM resources")
	iamClient := iam.New(config)

	_, err := iamClient.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: aws.String(cluster.instanceProfileName()),
		RoleName:            aws.String(cluster.roleName()),
	})
	if err != nil {
		log.Warnf("  error while removing role from instance profile: %s", err)
	}

	log.Infof("  instance profile %s", cluster.instanceProfileName())
	_, err = iamClient.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{
		InstanceProfileName: aws.String(cluster.instanceProfileName()),
	})
	if err != nil {
		log.Warnf("  error while deleting instance profile: %s", err)
	}

	// There must be a better way...but i couldn't find another way to look up the policy ARN.
	policies, err := iamClient.ListPolicies(&iam.ListPoliciesInput{
		Scope: aws.String("Local"),
	})
	for _, policy := range policies.Policies {
		if *policy.PolicyName == cluster.managerPolicyName() {
			_, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
				RoleName:  aws.String(cluster.roleName()),
				PolicyArn: policy.Arn,
			})
			if err != nil {
				log.Warnf("  error while detaching role policy: %s", err)
			}

			log.Infof("  policy %s", *policy.Arn)
			_, err = iamClient.DeletePolicy(&iam.DeletePolicyInput{
				PolicyArn: policy.Arn,
			})
			if err != nil {
				log.Warnf("  error while deleting policy: %s", err)
			}
		}
	}

	if err != nil {
		log.Warnf("  error while deleting role policy: %s", err)
	}

	log.Infof("  role %s", cluster.roleName())
	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: aws.String(cluster.roleName())})
	if err != nil {
		log.Warnf("  error while deleting IAM role: %s", err)
	}
}

func destroyNetwork(config client.ConfigProvider, cluster clusterID, vpcID string) {
	log.Info("Destroying network resources")
	ec2Client := ec2.New(config)

	securityGroups, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: cluster.resourceFilter(vpcID),
	})
	if err == nil {
		for _, securityGroup := range securityGroups.SecurityGroups {
			log.Infof("  security group %s", *securityGroup.GroupId)
			_, err = ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
				GroupId: securityGroup.GroupId,
			})
			if err != nil {
				log.Warnf("  error while deleting security group: %s", err)
			}
		}
	} else {
		log.Warnf("  error while describing security groups: %s", err)
	}

	subnets, err := ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{Filters: []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcID)},
		},
		cluster.clusterFilter(),
	}})
	if err == nil {
		for _, subnet := range subnets.Subnets {
			log.Infof("  subnet %s", *subnet.SubnetId)
			_, err = ec2Client.DeleteSubnet(&ec2.DeleteSubnetInput{SubnetId: subnet.SubnetId})
			if err != nil {
				log.Warnf("  error while deleting subnet: %s", err)
			}
		}
	} else {
		log.Warnf("  error while looking up subnets: %s", err)
	}

	internetGateways, err := ec2Client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			cluster.clusterFilter(),
		},
	})
	if err == nil {
		for _, internetGateway := range internetGateways.InternetGateways {
			log.Infof(
				"  detaching internet gateway %s from VPC %s",
				*internetGateway.InternetGatewayId,
				vpcID)
			_, err := ec2Client.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
				InternetGatewayId: internetGateway.InternetGatewayId,
				VpcId:             aws.String(vpcID),
			})
			if err != nil {
				log.Warnf("  error detaching internet gateways: %s", err)
			}

			log.Infof("  internet gateway %s", *internetGateway.InternetGatewayId)
			_, err = ec2Client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
				InternetGatewayId: internetGateway.InternetGatewayId,
			})
			if err != nil {
				log.Warnf("  error deleting internet gateways: %s", err)
			}
		}
	} else {
		log.Warnf("  error looking up internet gateways: %s", err)
	}

	routeTables, err := ec2Client.DescribeRouteTables(
		&ec2.DescribeRouteTablesInput{Filters: cluster.resourceFilter(vpcID)})
	if err == nil {
		for _, routeTable := range routeTables.RouteTables {
			log.Infof("  route table %s", *routeTable.RouteTableId)
			_, err = ec2Client.DeleteRouteTable(&ec2.DeleteRouteTableInput{
				RouteTableId: routeTable.RouteTableId,
			})
			if err != nil {
				log.Warnf("  error while deleting route table: %s", err)
			}
		}
	} else {
		log.Warnf("  error while describing route tables: %s", err)
	}

	log.Infof("  VPC %s", vpcID)
	_, err = ec2Client.DeleteVpc(&ec2.DeleteVpcInput{VpcId: aws.String(vpcID)})
	if err != nil {
		log.Warnf("  error while deleting VPC: %s", err)
	}
}

func destroy(cluster clusterID) error {
	sess := cluster.getAWSClient()
	ec2Client := ec2.New(sess)

	vpcs, err := ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{Filters: []*ec2.Filter{cluster.clusterFilter()}})
	if err != nil {
		return fmt.Errorf("Failed to look up VPC: %s", err)
	}

	// TODO(wfarner): We omit the VPC ID from resource tags and allow more failure-resistant cleanup as long as we
	// disallow clusters of the same name to exist within a region.
	var vpcID string
	switch len(vpcs.Vpcs) {
	case 0:
		log.Warnf("No VPCs found for cluster %s, unable to remove networks or instances", cluster.name)
	case 1:
		vpcID = *vpcs.Vpcs[0].VpcId
	default:
		log.Warnf(
			"Found multiple VPCs for cluster %s, unable to remove networks or instances",
			cluster.name)
	}

	if vpcID != "" {
		destroyInstances(sess, cluster, vpcID)
	}

	destroyEBSVolues(sess, cluster)

	destroyAccessRoles(sess, cluster)

	if vpcID != "" {
		destroyNetwork(sess, cluster, vpcID)
	}

	return nil
}
