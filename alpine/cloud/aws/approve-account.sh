#!/bin/bash

# Usage: ./aws/approve-account.sh [ACCOUNT_ID] [AMI_FILE]
#
# ACCOUNT_ID must be a valid AWS account ID
#
# AMI_FILE must be a newline-delimited file containing the AMI IDs to approve
# launch permissions for the given account and their region, e.g.:
# 
# ami-xxxxxxx us-west-1
# ami-yyyyyyy us-east-1

set -e

source "cloud/build-common.sh"
source "cloud/aws/common.sh"

USER_ID="$1"

if [ ${#USER_ID} -lt 12 ]
then
	# Pad zeros in front so it will always be 12 chars long, e.g. some AWS
	# accounts have ID like '123123123123' and others like '000123123123'
	USER_ID_PADDED=$(printf "%0$((12-${#USER_ID}))d%s" 0 ${USER_ID})
else
	USER_ID_PADDED="${USER_ID}"
fi

AMI_FILE="$2"

if [ ! -f ${AMI_FILE} ]
then
	errecho "AMI file not found."
	exit 1
fi

while read REGION_AMI_ID
do
	REGION=$(echo ${REGION_AMI_ID} | cut -d' ' -f 1)
	IMAGE_ID=$(echo ${REGION_AMI_ID} | cut -d' ' -f 2)
	arrowecho "Approving launch for ${IMAGE_ID} in ${REGION}"
	aws ec2 modify-image-attribute \
		--region ${REGION} \
		--image-id ${IMAGE_ID} \
		--launch-permission "{
			\"Add\": [{
				\"UserId\": \"${USER_ID_PADDED}\"
			}]
		}"
done <${AMI_FILE}

arrowecho "Done approving account ${USER_ID_PADDED}"
