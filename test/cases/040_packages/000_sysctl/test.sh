#!/bin/sh
# SUMMARY: Check that the sysctl config works
# LABELS:
# REPEAT:
# AUTHOR: Justin Cormack <justin.cormack@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -iname "test-sysctl*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build test-sysctl
RESULT="$(linuxkit run qemu -kernel test-sysctl)"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
