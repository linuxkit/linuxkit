#!/bin/sh
# SUMMARY: Test build and insertion of kernel modules
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

IMAGE_NAME="kmod-test"

clean_up() {
	docker rmi ${IMAGE_NAME} || true
	find . -iname "kmod*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Make sure we have the latest kernel image
docker pull linuxkit/kernel:4.9.x
# Build a package
docker build -t ${IMAGE_NAME} .
# Build a LinuxKit image with kernel module (and test script)
moby build -output kernel+initrd kmod
# Run it
linuxkit run qemu kmod | grep -q "Hello LinuxKit"
