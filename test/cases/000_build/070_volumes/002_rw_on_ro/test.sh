#!/bin/sh
# SUMMARY: Check that writing to a volume makes it visible
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=volume_rw_on_r0

clean_up() {
	rm -rf ${NAME}-* ${DISK}
}
trap clean_up EXIT

# Test code goes here

set +e
linuxkit build --format kernel+initrd --name ${NAME} test.yml
retcode=$?
set -e

# the build should fail, as we have are mounting a read-only volume as read-write in a container
if [ $retcode -eq 0 ]; then
	exit 1
fi
