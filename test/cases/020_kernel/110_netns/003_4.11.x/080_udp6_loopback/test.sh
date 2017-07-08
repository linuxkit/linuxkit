#!/bin/sh
# SUMMARY: Network namespace stress test with UDP/IPv6 over the loopback interface
# LABELS: kernel-extra
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -depth -iname "test-netns*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build -output kernel+initrd test-netns.yml
RESULT="$(linuxkit run -cpus 2 test-netns)"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
