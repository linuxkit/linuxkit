#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible after leveraging input tar
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check_input_tar_conflict_filename

clean_up() {
	rm -f ${NAME}-*.tar
}

trap clean_up EXIT

logfile=$(mktemp)

# do not include the sbom, because the SBoM unique IDs per file/package are *not* deterministic,
# (currently based upon syft), and thus will make the file non-reproducible

# the first one should build normally without a problem
linuxkit build --no-sbom --format tar --o "${NAME}-1.tar" ./test.yml

# second one should fail because the input tar has the same filename as the output tar
set +e
linuxkit build -v --no-sbom --format tar --input-tar "${NAME}-1.tar" --o "${NAME}-1.tar" ./test.yml 2>&1
ret="$?"
set -e

if [ "$ret" -eq 0 ]; then
	echo "Expected the build to fail, but it succeeded"
	exit 1
fi

exit 0
