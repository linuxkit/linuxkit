#!/bin/sh
# SUMMARY: Check that the userdata is found and read when on a cidata partition
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=metadata
DISK=disk.img

clean_up() {
	rm -rf ${NAME}-* ${DISK}
}
trap clean_up EXIT

# Test code goes here

# generate our cdrom image
ISOFILE=/tmp/cidata.iso

docker run -i --rm -v $(pwd)/geniso.sh:/geniso.sh:ro alpine:3.13 /geniso.sh > ${ISOFILE}

linuxkit build --format kernel+initrd --name ${NAME} test.yml
RESULT="$(linuxkit run --disk file=${DISK},size=32M --disk file=${ISOFILE} ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
