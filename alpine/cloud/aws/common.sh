#!/bin/sh

set -e

TAG_KEY=${TAG_KEY:-editionsdev}
TAG_KEY_PREV=${TAG_KEY_PREV:-editionsdev}

INSTANCE_ENDPOINT="http://169.254.169.254/latest"
INSTANCE_METADATA_API_ENDPOINT="${INSTANCE_ENDPOINT}/meta-data/"
IMAGE_NAME=${IMAGE_NAME:-"Moby Linux ${TAG_KEY}"}
IMAGE_DESCRIPTION=${IMAGE_DESCRIPTION:-"The best OS for running Docker, version ${TAG_KEY}"}

current_instance_region()
{
    curl -s "${INSTANCE_ENDPOINT}/dynamic/instance-identity/document" | jq .region -r
}

current_instance_az()
{
    curl -s "${INSTANCE_METADATA_API_ENDPOINT}/placement/availability-zone"
}

current_instance_id()
{
    curl -s "${INSTANCE_METADATA_API_ENDPOINT}/instance-id"
}

# We tag resources created as part of the build to ensure that they can be
# cleaned up later.
tag()
{
    arrowecho "Tagging $1 with ${TAG_KEY}, $2, and $3"
    aws ec2 create-tags --resources "$1" --tags "Key=${TAG_KEY},Value=" >/dev/null
    aws ec2 create-tags --resources "$1" --tags "Key=date,Value=$2" >/dev/null
    aws ec2 create-tags --resources "$1" --tags "Key=channel,Value=$3" >/dev/null
}
