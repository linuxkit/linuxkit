#!/bin/sh
# SUMMARY: Test the Docker for Mac blueprint
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=docker-for-mac

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${IMAGE_NAME}*
}

trap clean_up EXIT

# Test code goes here
moby build "${LINUXKIT_BLUEPRINTS_DIR}/${IMAGE_NAME}.yml" 

exit 0

