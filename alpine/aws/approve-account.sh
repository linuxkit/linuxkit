#!/bin/bash

# Usage: ./aws/approve-account [ACCOUNT_ID] [AMI_FILE]
#
# ACCOUNT_ID must be a valid AWS account ID
#
# AMI_FILE must be a newline-delimited file containing the AMI IDs to approve
# launch permissions for the given account

set -e

source ./aws/common.sh

AMIS=($AMIS)
USER_ID="$1"
AMI_FILE="$2"

while read REGION_AMI_ID; do
    aws ec2 modify-image-attribute --image-id ${REGION_AMI_ID} --launch-permission "{
        \"Add\": [{
            \"UserId\": \"${USER_ID}\"
        }]
    }"
done <${AMI_FILE}
