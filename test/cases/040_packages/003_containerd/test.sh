#!/bin/sh
# SUMMARY: Run containerd test
# disable containerd test because of: https://github.com/containerd/containerd/issues/2447
# LABELS: disabled
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=containerd

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Test code goes here
linuxkit build -format kernel+initrd -name "${NAME}" test.yml
RESULT="$(linuxkit run -mem 2048 -disk size=2G ${NAME})"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
