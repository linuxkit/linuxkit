#!/bin/sh
# SUMMARY: Run containerd test
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=cgroupv2

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Test code goes here
linuxkit build --format kernel+initrd --name "${NAME}" test.yml
RESULT="$(linuxkit run --mem 3072 ${NAME})"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
