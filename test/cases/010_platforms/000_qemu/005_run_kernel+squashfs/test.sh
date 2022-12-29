#!/bin/sh
# SUMMARY: Check that the kernel+squashfs image boots in qemu
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-squashfs

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

linuxkit build --format kernel+squashfs --name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-squashfs.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
linuxkit run qemu --squashfs "${NAME}" | grep -q "Welcome to LinuxKit"

exit 0
