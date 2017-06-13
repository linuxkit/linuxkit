#!/bin/sh
# SUMMARY: Check that the sysctl config works
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -depth -iname "test-sysctl*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build -output kernel+initrd test-sysctl.yml
RESULT="$(linuxkit run test-sysctl)"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
