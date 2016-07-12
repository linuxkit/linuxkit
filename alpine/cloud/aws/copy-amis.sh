#!/bin/bash

# Usage: ./cloud/aws/copy-amis.sh 
# Assumptions: 
# - A finished Moby AMI ID has been deposited in ./cloud/aws/ami_id.out.  (This is
#   the behavior of the ./cloud/aws/bake-ami.sh script)
# - (recommended) IMAGE_NAME is set to a custom value (e.g., "Moby Linux")
# - (recommended) IMAGE_DESCRIPTION is set to a custom value (e.g., "1.12-0-tp1")
#
# 
# Outputs: 
# - A file of newline delimited AMI IDs representing the AMI for each region.
# - A file containing a subsection of a CloudFormation template outlining these AMIs (JSON).

set -e

source "cloud/build-common.sh"
source "cloud/aws/common.sh"

SOURCE_AMI_ID=$(cat ./cloud/aws/ami_id.out)

# To have a list of just the IDs (approve accounts later if desired)
AMIS_IDS_DEST="./cloud/aws/copied_image_regions_${SOURCE_AMI_ID}.out"

# File to drop the (mostly correct) CF template section in
CF_TEMPLATE="./cloud/aws/cf_image_regions_${SOURCE_AMI_ID}.out"

cfecho () {
    echo "$@" >>${CF_TEMPLATE}
}

cfprintf () {
    printf "$@" >>${CF_TEMPLATE}
}

if [[ -f ${AMIS_IDS_DEST} ]]; then
    rm ${AMIS_IDS_DEST}
fi

if [[ -f ${CF_TEMPLATE} ]]; then
    rm ${CF_TEMPLATE}
fi

cfecho '"AWSRegionArch2AMI": {'

REGIONS=(us-west-1 us-west-2 us-east-1 eu-west-1 eu-central-1 ap-southeast-1 ap-northeast-1 ap-southeast-2 ap-northeast-2 sa-east-1)

for REGION in ${REGIONS[@]}; do
    REGION_AMI_ID=$(aws ec2 copy-image \
        --source-region $(current_instance_region) \
        --source-image-id "${SOURCE_AMI_ID}"  \
        --region "${REGION}" \
        --name "${IMAGE_NAME}" \
        --description "${IMAGE_DESCRIPTION}" | jq -r .ImageId)

    echo "${REGION_AMI_ID}"

    echo "${REGION} ${REGION_AMI_ID}" >>${AMIS_IDS_DEST}

    cfprintf "    \"${REGION}\": {
        \"HVM64\": \"${REGION_AMI_ID}\",
        \"HVMG2\": \"NOT_SUPPORTED\"
    }"

    # Emit valid JSON.  No trailing comma on last element.
    if [[ ${REGION} != ${REGIONS[-1]} ]]; then
        cfecho ","
    else
        cfecho
    fi
done

cfecho "}"

echo "All done.  The results for adding to CloudFormation can be"
echo "viewed here:"
arrowecho ${CF_TEMPLATE}
echo
echo "The plain list of AMIs can be viewed here:"
arrowecho ${AMIS_IDS_DEST}
