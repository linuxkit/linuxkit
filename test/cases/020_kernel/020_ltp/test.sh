#!/bin/sh
# SUMMARY: Run the Linux Testing Project tests
# LABELS: slow, gcp
# REPEAT:
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=ltp

clean_up() {
	rm -rf ${IMAGE_NAME}*
}
trap clean_up EXIT

# Test code goes here
moby build --name ${IMAGE_NAME} test.yml
# NOTE: It's pushed using the CLOUDSDK_IMAGE_NAME
linuxkit push gcp ${IMAGE_NAME}.img.tar.gz
RESULT="$(linuxkit run gcp -skip-cleanup -machine n1-highcpu-4 ${CLOUDSDK_IMAGE_NAME})"
echo "${RESULT}" | grep -q "suite has passed"

exit 0
