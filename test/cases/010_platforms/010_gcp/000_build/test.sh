#!/bin/sh
# SUMMARY: Test building an image for GCP
# LABELS: build
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=gcp

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${IMAGE_NAME}*
}

trap clean_up EXIT

# Test code goes here
moby build -name "${IMAGE_NAME}" test.yml
[ -f "${IMAGE_NAME}.img.tar.gz" ] || exit 1
# As build and run on different machines, copy to the artifacts directory
cp -f "${IMAGE_NAME}.img.tar.gz" "${LINUXKIT_ARTIFACTS_DIR}/test.img.tar.gz"

exit 0
