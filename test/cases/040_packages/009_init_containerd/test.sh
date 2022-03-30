#!/bin/sh
# SUMMARY: Check that the containerd command-line options have been enabled by init
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"
NAME=init_containerd

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Test code goes here
linuxkit build -format kernel+initrd -name "${NAME}" test.yml
RESULT="$(linuxkit run $NAME)"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
