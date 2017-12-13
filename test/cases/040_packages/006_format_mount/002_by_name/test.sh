#!/bin/sh
# SUMMARY: Check that a formatted disk can be mounted by name
# Disabled on arm64: https://github.com/linuxkit/linuxkit/issues/2808
# LABELS: amd64
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-format
DISK=disk.img

clean_up() {
	rm -rf ${NAME}-* ${DISK} test.yml
}
trap clean_up EXIT

if [ "${RT_OS}" = "osx" ]; then
	DEVICE="/dev/vda"
else
	DEVICE="/dev/sda"
fi

sed -e "s,@DEVICE@,${DEVICE},g" test.yml.in > test.yml
linuxkit build -format kernel+initrd -name ${NAME} test.yml
RESULT="$(linuxkit run -disk file=${DISK},size=512M ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
