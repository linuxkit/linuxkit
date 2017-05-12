#!/bin/sh
# SUMMARY: Test running an image with qemu
# LABELS:
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=test-qemu-build

clean_up() {
	# remove any files, containers, images etc
	rm -rf "${LINUXKIT_TMPDIR:?}/${IMAGE_NAME:?}*" || true
}

trap clean_up EXIT

# Test code goes here
[ -f "${LINUXKIT_TMPDIR}/${IMAGE_NAME}-kernel" ] || exit 1
linuxkit run qemu "${LINUXKIT_TMPDIR}/${IMAGE_NAME}" | grep -q "Welcome to LinuxKit"
exit 0
