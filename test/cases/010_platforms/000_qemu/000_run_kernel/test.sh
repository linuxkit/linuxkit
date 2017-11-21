#!/bin/sh
# SUMMARY: Check that the kernel+initrd image boots in qemu
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-kernel

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

linuxkit build -format kernel+initrd -name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
linuxkit run qemu -kernel "${NAME}" | grep -q "Welcome to LinuxKit"

exit 0
