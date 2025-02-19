#!/bin/sh
# SUMMARY: Check that writing to a volume makes it visible
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=volume_rw_on_rw

clean_up() {
	rm -rf ${NAME}-* ${DISK}
}
trap clean_up EXIT

# Test code goes here

linuxkit build --format kernel+initrd --name ${NAME} test.yml
RESULT="$(linuxkitrun ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
