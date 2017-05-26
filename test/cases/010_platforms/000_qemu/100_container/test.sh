#!/bin/sh
# SUMMARY: Check that qemu runs containerised
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-kernel

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${NAME}* || true
}

trap clean_up EXIT

# check if qemu is installed locally
QEMU=$(command -v qemu-system-x86_64 || true)
if [ -z "${QEMU}" ]; then
    # No qemu installed so don't bother to test as all the other
    # qemu tests would have been run containerised
    echo "No locally installed qemu"
    exit $RT_CANCEL
fi

moby build -output kernel+initrd -name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
linuxkit run qemu -containerized "${NAME}" | grep -q "Welcome to LinuxKit"
exit 0
