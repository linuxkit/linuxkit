#!/bin/bash

set -e

TAG_KEY=moby-image
INSTANCE_ENDPOINT=http://169.254.169.254/latest
INSTANCE_METADATA_API_ENDPOINT=${INSTANCE_ENDPOINT}/meta-data/
IMAGE_NAME=${IMAGE_NAME:-"Moby Linux"}
IMAGE_DESCRIPTION=${IMAGE_DESCRIPTION:-"The best OS for running Docker"}

function arrowecho () {
    echo " --->" "$@"
}

function current_instance_region () {
    curl -s ${INSTANCE_ENDPOINT}/dynamic/instance-identity/document | jq .region -r
}

function current_instance_az () {
    curl -s ${INSTANCE_METADATA_API_ENDPOINT}/placement/availability-zone
}

function current_instance_id () {
    curl -s ${INSTANCE_METADATA_API_ENDPOINT}/instance-id
}

# We tag resources created as part of the build to ensure that they can be
# cleaned up later.
function tag () {
    arrowecho "Tagging $1"
    aws ec2 create-tags --resources "$1" --tags Key=${TAG_KEY},Value= >/dev/null
}
