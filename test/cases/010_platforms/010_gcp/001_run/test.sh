#!/bin/sh
# SUMMARY: Test running an image with qemu
# LABELS: gcp
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=test

clean_up() {
	# remove any files, containers, images etc
	echo "Nothing to cleanup..."
}

trap clean_up EXIT

# Test code goes here
[ -f "${LINUXKIT_ARTIFACTS_DIR}/${IMAGE_NAME}.img.tar.gz" ] || exit 1
linuxkit run gcp "${LINUXKIT_ARTIFACTS_DIR}/${IMAGE_NAME}" | grep -q "Welcome to LinuxKit"
exit 0
