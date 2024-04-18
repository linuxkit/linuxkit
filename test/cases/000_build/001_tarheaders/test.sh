#!/bin/sh
# SUMMARY: Check that tar output format build contains proper headers for each file
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=tarheaders

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

# do not include the sbom, because the SBoM unique IDs per file/package are *not* deterministic,
# (currently based upon syft), and thus will make the file non-reproducible
linuxkit build --no-sbom --format tar --name "${NAME}" ./test.yml

# Check that the tarball contains the expected headers
# see that python is installed
PYTHON=
if which python ; then PYTHON=python ; elif which python3; then PYTHON=python3 ; else
echo "Failed to find any executable python or python3"
exit 1
fi
FAILED=$(python ./tarheaders.py "${NAME}.tar")

if [ -n "${FAILED}" ]; then
	echo "Failed to find linuxkit.packagesource headers for the following files:"
	echo "${FAILED}"
	exit 1
fi

exit 0
