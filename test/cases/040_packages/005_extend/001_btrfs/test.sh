#!/bin/sh
# SUMMARY: Check that a btrfs partition can be extended
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME_CREATE=extend-btrfs
NAME_EXTEND=extend-btrfs
DISK=disk.img

clean_up() {
	rm -rf ${NAME_CREATE}-* ${NAME_EXTEND}-* ${DISK}
}
trap clean_up EXIT

# Test code goes here
linuxkit build -name "${NAME_CREATE}" -format kernel+initrd test-create.yml
linuxkit run -disk file="${DISK}",format=raw,size=256M "${NAME_CREATE}"
[ -f "${DISK}" ] || exit 1
# osx takes issue with bs=1M
dd if=/dev/zero bs=1048576 count=256 >> "${DISK}"
linuxkit build -format kernel+initrd -name ${NAME_EXTEND} test.yml
RESULT="$(linuxkit run -disk file=${DISK} ${NAME_EXTEND})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
