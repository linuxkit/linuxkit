#!/bin/sh
# SUMMARY: Check that gcp output format is generated
# LABELS: amd64

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

linuxkit build --format gcp --name "${NAME}" ../test.yml
[ -f "${NAME}.img.tar.gz" ] || exit 1

exit 0
