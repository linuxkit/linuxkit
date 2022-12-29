#!/bin/sh
# SUMMARY: Check that the losetup package works
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=dm-crypt
DISK=disk.img

clean_up() {
        rm -rf ${NAME}-* ${DISK}
}
trap clean_up EXIT

# Test code goes here
linuxkit build --format kernel+initrd --name ${NAME} test.yml
RESULT="$(linuxkit run --disk file=${DISK},size=20M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
