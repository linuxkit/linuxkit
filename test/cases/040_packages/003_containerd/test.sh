#!/bin/sh
# SUMMARY: Run containerd test
# skipping while status of go 1.8 support in containerd is unsure
# https://github.com/containerd/containerd/issues/1632
# LABELS: skip
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
moby build -format kernel+initrd -name "${NAME}" test.yml
RESULT="$(linuxkit run -mem 2048 -disk size=2G ${NAME})"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
