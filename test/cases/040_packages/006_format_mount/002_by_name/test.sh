#!/bin/sh
# SUMMARY: Check that a formatted disk can be mounted by name
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-format

clean_up() {
	find . -depth -iname "${NAME}*" -not -iname "*.yml" -exec rm -rf {} \;
	rm -rf test.yml || true
}

trap clean_up EXIT
# Test code goes here
if [ "${RT_OS}" = "osx" ]; then
	DEVICE="/dev/vda"
else
	DEVICE="/dev/sda"
fi

sed -e "s,@DEVICE@,${DEVICE},g" test.yml.in > test.yml
moby build -name ${NAME} -output kernel+initrd test.yml
RESULT="$(linuxkit run -disk file=${NAME}1.img,size=512M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
