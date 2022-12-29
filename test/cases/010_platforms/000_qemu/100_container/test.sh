#!/bin/sh
# SUMMARY: Check that qemu runs containerised
# LABELS: skip

# this test is not working at present see https://github.com/linuxkit/linuxkit/issues/2020

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-kernel

clean_up() {
	rm -rf ${NAME}*
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

linuxkit build --format kernel+initrd --name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
linuxkit run qemu -containerized "${NAME}" | grep -q "Welcome to LinuxKit"

exit 0
