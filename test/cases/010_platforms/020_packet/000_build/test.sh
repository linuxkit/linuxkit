#!/bin/sh
# SUMMARY: Test building an image for packet.net
# LABELS: build
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=packet

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${IMAGE_NAME}*
}

trap clean_up EXIT

# Test code goes here
moby build --name "${IMAGE_NAME}" test.yml 
[ -f "${IMAGE_NAME}-kernel" ] || exit 1

# As build and run on different machines, copy to the artifacts directory
find . -iname "${IMAGE_NAME}-*" -exec cp {} "${LINUXKIT_ARTFACTS_DIR}/{}" \;
exit 0
