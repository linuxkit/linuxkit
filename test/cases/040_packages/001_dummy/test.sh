#!/bin/sh
# SUMMARY: Check that the dummy pkg exists
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"
NAME=dummy

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Test code goes here
linuxkit build --format kernel+initrd --name "${NAME}" test.yml
# all we are checking is that we can build using the dummy package, which
# was not pushed out

exit 0
