#!/bin/bash

# Usage: ./aws/approve-account.sh [ACCOUNT_ID] [AMI_FILE]
#
# ACCOUNT_ID must be a valid AWS account ID
#
# AMI_FILE must be a newline-delimited file containing the AMI IDs to approve
# launch permissions for the given account

set -e

source ./aws/common.sh

USER_ID="$1"
AMI_FILE="$2"

while read REGION_AMI_ID; do
    REGION=$(echo ${REGION_AMI_ID} | cut -d' ' -f 1)
    IMAGE_ID=$(echo ${REGION_AMI_ID} | cut -d' ' -f 2)
    arrowecho "Approving launch for ${IMAGE_ID} in ${REGION}"
    aws ec2 modify-image-attribute \
        --region ${REGION} \
        --image-id ${IMAGE_ID} \
        --launch-permission "{
            \"Add\": [{
                \"UserId\": \"${USER_ID}\"
            }]
        }"
done <${AMI_FILE}

arrowecho "Done approving account ${USER_ID}"
