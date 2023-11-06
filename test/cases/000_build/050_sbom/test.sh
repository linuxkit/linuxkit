#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=sbom

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

# build the packages we need
linuxkit pkg build ./package1 ./package2

# build the image we need
linuxkit build --format tar --name "${NAME}" ./test.yml

# check that we got the SBoM
tar -tvf ${NAME}.tar sbom.spdx.json

exit 0
