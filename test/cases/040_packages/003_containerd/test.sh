#!/bin/sh
# SUMMARY: Run containerd test
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -depth -iname "test-containerd*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build test-containerd.yml
RESULT="$(linuxkit run -mem 2048 -disk size=2G test-containerd)"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
