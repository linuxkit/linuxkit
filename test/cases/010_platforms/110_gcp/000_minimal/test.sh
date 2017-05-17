#!/bin/sh
# SUMMARY: Test that a minimal image boots on GCP
# LABELS:
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=gcp-minimal

clean_up() {
	rm -rf ${IMAGE_NAME}*
}

trap clean_up EXIT

# Test code goes here
moby build -name ${IMAGE_NAME} test.yml
linuxkit push gcp ${IMAGE_NAME}.img.tar.gz
RESULT=$(linuxkit run gcp "${CLOUDSDK_IMAGE_NAME}")
echo "$RESULT"| grep -q "I Can Haz Linux?"
exit 0
