#!/bin/sh
# SUMMARY: Check that the binfmt package works
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -depth -iname "test-binfmt*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build -output kernel+initrd test-binfmt.yml
RESULT="$(linuxkit run test-binfmt)"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
