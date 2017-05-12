#!/bin/sh
# SUMMARY: Test building an image for qemu
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
[ -f "${IMAGE_NAME}-kernel" ] || exit 1
[ -f "${IMAGE_NAME}-initrd.img" ] || exit 1
[ -f "${IMAGE_NAME}-cmdline" ]|| exit 1

find . -iname "${IMAGE_NAME}-*" -exec cp {} "${LINUXKIT_TMPDIR}/{}" \;
exit 0
