#!/bin/sh
# SUMMARY: Check that an XFS formatted disk can be mounted
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-format
DISK=disk.img

clean_up() {
	rm -rf ${NAME}-* ${DISK}
}
trap clean_up EXIT

linuxkit build -format kernel+initrd -name ${NAME} test.yml
RESULT="$(linuxkit run -disk file=${DISK},size=512M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
