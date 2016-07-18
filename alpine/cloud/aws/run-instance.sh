#!/bin/bash

# Quick script to boot an instance from generated AMI.  Intended to be invoked
# from "alpine" directory.

set -e

JOINERS_COUNT=${JOINERS_COUNT:-1}
METADATA="http://169.254.169.254/latest/meta-data"
MANAGER_SG="docker-swarm-ingress"

manager_sg_id()
{
	aws ec2 describe-security-groups \
		--filter Name=group-name,Values=${MANAGER_SG} | jq -r .SecurityGroups[0].GroupId
}

attach_security_group()
{
	MANAGER_SG_ID=$(manager_sg_id)
	if [ ${MANAGER_SG_ID} == "null" ]
	then
		CUR_INSTANCE_MAC=$(wget -qO- ${METADATA}/network/interfaces/macs)
		CUR_INSTANCE_VPC_CIDR=$(wget -qO- ${METADATA}/network/interfaces/macs/${CUR_INSTANCE_MAC}vpc-ipv4-cidr-block)
		MANAGER_SG_ID=$(aws ec2 create-security-group \
			--group-name ${MANAGER_SG} \
			--description "Allow inbound access to Docker API and for remote join node connection" | jq -r .GroupId)

		echo "Created security group ${MANAGER_SG_ID}"

		# Hack to wait for SG to be created before adding rules
		sleep 5

		# For Docker API
		aws ec2 authorize-security-group-ingress \
			--group-id ${MANAGER_SG_ID} \
			--protocol tcp \
			--port 2375 \
			--cidr ${CUR_INSTANCE_VPC_CIDR}

		# For Swarm join node connection
		aws ec2 authorize-security-group-ingress \
			--group-id ${MANAGER_SG_ID} \
			--protocol tcp \
			--port 4500 \
			--cidr ${CUR_INSTANCE_VPC_CIDR}
	fi

	aws ec2 modify-instance-attribute \
		--instance-id "$1" \
		--groups ${MANAGER_SG_ID}
}

poll_instance_log()
{
	echo "Waiting for instance boot log to become available"

	INSTANCE_BOOT_LOG="null"
	while [ ${INSTANCE_BOOT_LOG} == "null" ]
	do
		INSTANCE_BOOT_LOG=$(aws ec2 get-console-output --instance-id "$1" | jq -r .Output)
		sleep 5
	done

	aws ec2 get-console-output --instance-id "$1" | jq -r .Output
}

OLD_INSTANCE_IDS=$(cat ./cloud/aws/instance_id.out | tr '\n' ' ')
aws ec2 terminate-instances --instance-id ${OLD_INSTANCE_IDS} || true

if [ ! -f ./cloud/aws/ami_id.out ]
then
	echo "AMI ID to launch instance from not found"
	exit 1
fi

AMI_ID=$(cat ./cloud/aws/ami_id.out)

echo "Using image ${AMI_ID}"

MANAGER_INSTANCE_ID=$(aws ec2 run-instances \
	--image-id ${AMI_ID} \
	--instance-type t2.micro \
	--user-data file://./cloud/aws/manager-user-data.sh | jq -r .Instances[0].InstanceId)

aws ec2 create-tags --resources ${MANAGER_INSTANCE_ID} --tags Key=Name,Value=$(whoami)-docker-swarm-manager

echo "Running manager instance ${MANAGER_INSTANCE_ID}"

# Deliberately truncate file here.
echo ${MANAGER_INSTANCE_ID} >./cloud/aws/instance_id.out

attach_security_group ${MANAGER_INSTANCE_ID}

# User can set this variable to indicate they want a whole swarm.
if [ ! -z "$JOIN_INSTANCES" ]
then
	MANAGER_IP=$(aws ec2 describe-instances \
		--instance-id ${MANAGER_INSTANCE_ID} | jq -r .Reservations[0].Instances[0].NetworkInterfaces[0].PrivateIpAddresses[0].PrivateIpAddress)

	TMP_JOINER_USERDATA=/tmp/joiner-user-data-${MANAGER_INSTANCE_ID}.sh

	cat ./cloud/aws/joiner-user-data.sh | sed "s/{{MANAGER_IP}}/${MANAGER_IP}/" >${TMP_JOINER_USERDATA}

	JOINER_INSTANCE_IDS=$(aws ec2 run-instances \
		--image-id ${AMI_ID} \
		--instance-type t2.micro \
		--count ${JOINERS_COUNT} \
		--user-data file://${TMP_JOINER_USERDATA} | jq -r .Instances[].InstanceId)

	echo "Joining nodes:" ${JOINER_INSTANCE_IDS}

	NODE_NUMBER=0

	for ID in ${JOINER_INSTANCE_IDS}
	do
		echo "Tagging joiner instance #${NODE_NUMBER}: ${ID}"

		# For debugging purposes only.  In "production" this SG should not be
		# attached to these instances.
		attach_security_group ${ID}

		# Do not truncate file here.
		echo ${ID} >>./cloud/aws/instance_id.out

		# TODO: Get list of ids and do this for each if applicable.
		aws ec2 create-tags --resources ${ID} --tags Key=Name,Value=$(whoami)-docker-swarm-joiner-${NODE_NUMBER}

		NODE_NUMBER=$((NODE_NUMBER+1))
	done

	exit
fi

echo "Waiting for manager to be running..."
aws ec2 wait instance-running --instance-ids $(cat ./cloud/aws/instance_id.out | tr '\n' ' ')

poll_instance_log ${MANAGER_INSTANCE_ID}
