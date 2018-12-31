#!/bin/sh
# SUMMARY: Test build and insertion of kernel modules
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=kmod
IMAGE_NAME=kmod-test

clean_up() {
	docker rmi ${IMAGE_NAME} || true
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Make sure we have the latest kernel image
docker pull linuxkit/kernel:4.9.148
# Build a package
docker build -t ${IMAGE_NAME} .

# Build and run a LinuxKit image with kernel module (and test script)
linuxkit build -format kernel+initrd -name "${NAME}" test.yml
RESULT="$(linuxkit run ${NAME})"
echo "${RESULT}" | grep -q "Hello LinuxKit"

exit 0
