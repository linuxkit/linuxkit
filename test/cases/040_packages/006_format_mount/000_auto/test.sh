#!/bin/sh
# SUMMARY: Check that a disk can be formatted and mounted
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-format

clean_up() {
	find . -depth -iname "${NAME}*" -not -iname "*.yml" -exec rm -rf {} \;
}

trap clean_up EXIT

# Test code goes here
moby build -name ${NAME} -output kernel+initrd test.yml
RESULT="$(linuxkit run -disk file=${NAME}1.img,size=512M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
