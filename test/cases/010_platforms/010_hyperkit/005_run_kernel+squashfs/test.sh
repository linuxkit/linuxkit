#!/bin/sh
# SUMMARY: Check that the kernel+squashfs image boots on hyperkit
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=hyperkit-squashfs

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

linuxkit build --format kernel+squashfs --name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-squashfs.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
./test.exp

exit 0
