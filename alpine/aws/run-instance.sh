#!/bin/bash

# Quick script to boot an instance from generated AMI.  Intended to be invoked
# from "alpine" directory.

set -e

INSTANCE_ID=$(cat ./aws/instance_id.out)
aws ec2 terminate-instances --instance-id ${INSTANCE_ID} || true

if [[ ! -f ./aws/ami_id.out ]]; then
    echo "AMI ID to launch instance from not found"
    exit 1
fi

AMI_ID=$(cat ./aws/ami_id.out)

echo "Running instance from ${AMI_ID}"

INSTANCE_ID=$(aws ec2 run-instances \
    --image-id ${AMI_ID} \
    --instance-type t2.nano \
    --user-data file://./aws/bootstrap.sh | jq -r .Instances[0].InstanceId)

aws ec2 create-tags --resources ${INSTANCE_ID} --tags Key=Name,Value=moby-boot-from-ami

echo "Running instance ${INSTANCE_ID}"
echo ${INSTANCE_ID} >./aws/instance_id.out

echo "Waiting for instance boot log to become available"

INSTANCE_BOOT_LOG="null"
while [[ ${INSTANCE_BOOT_LOG} == "null" ]]; do
    INSTANCE_BOOT_LOG=$(aws ec2 get-console-output --instance-id ${INSTANCE_ID} | jq -r .Output)
    sleep 5
done

aws ec2 get-console-output --instance-id ${INSTANCE_ID} | jq -r .Output
