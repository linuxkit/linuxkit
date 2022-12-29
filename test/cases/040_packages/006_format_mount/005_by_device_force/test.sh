#!/bin/sh
# SUMMARY: Check that the format and mount packages work
# Disabled on arm64: https://github.com/linuxkit/linuxkit/issues/2808
# LABELS: amd64
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-format
DISK1=disk1.img
DISK2=disk2.img

clean_up() {
	rm -rf ${NAME}-* ${DISK1} ${DISK2}
}
trap clean_up EXIT

linuxkit build --format kernel+initrd --name ${NAME} test.yml
RESULT="$(linuxkit run --disk file=${DISK1},size=512M --disk file=${DISK2},size=512M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
