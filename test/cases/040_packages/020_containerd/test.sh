#!/bin/sh
# SUMMARY: Check that the sysctl config works
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -iname "test-containerd*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build test-containerd
RESULT="$(linuxkit run qemu -mem 2048 test-containerd)"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
