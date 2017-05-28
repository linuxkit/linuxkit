#!/bin/sh
# SUMMARY: Check that the ca-certificates package works
# LABELS:
# REPEAT:

set -e
set -v

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -iname "test-ca-certificates*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build -output kernel+initrd test-ca-certificates
RESULT="$(linuxkit run qemu -kernel test-ca-certificates)"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
