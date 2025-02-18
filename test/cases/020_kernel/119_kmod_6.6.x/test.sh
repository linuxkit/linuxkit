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
docker pull linuxkit/kernel:6.6.71-bbe6930a9db6e1062d92357df654acc1d2d5832f
# Build a package
docker build -t ${IMAGE_NAME} .

# Build and run a LinuxKit image with kernel module (and test script)
linuxkit build --docker --format kernel+initrd --name "${NAME}" test.yml
RESULT="$(linuxkitrun ${NAME})"
echo "${RESULT}" | grep -q "Hello LinuxKit"

exit 0
