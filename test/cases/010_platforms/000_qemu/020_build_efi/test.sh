#!/bin/sh
# SUMMARY: Test building a UEFI image for qemu
# LABELS: build
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME=test-qemu-build

clean_up() {
	# remove any files, containers, images etc
	find . -iname "${IMAGE_NAME}*" -not -iname "*.yml" -exec rm {} \;
}

trap clean_up EXIT

# Test code goes here
moby build -name "${IMAGE_NAME}" test.yml
[ -f "${IMAGE_NAME}-efi.iso" ] || exit 1
cp "${IMAGE_NAME}-efi.iso" "${LINUXKIT_TMPDIR}/"
exit 0
