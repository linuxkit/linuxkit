#!/bin/sh
# SUMMARY: Run the Linux Testing Project tests 
# LABELS: slow, gcp
# REPEAT:
# AUTHOR: Dave Tucker <dt@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -iname "test-ltp*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

# Test code goes here
moby build test-ltp
linuxkit push test-ltp.img.tar.gz
RESULT="$(linuxkit run gcp -skip-cleanup -machine n1-highcpu-4 test-ltp)"
echo "${RESULT}" | grep -q "suite has passed"

exit 0
