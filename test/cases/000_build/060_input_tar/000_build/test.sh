#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible after leveraging input tar
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check_input_tar

clean_up() {
	rm -f ${NAME}-*.tar
}

trap clean_up EXIT

logfile=$(mktemp)

# do not include the sbom, because the SBoM unique IDs per file/package are *not* deterministic,
# (currently based upon syft), and thus will make the file non-reproducible
linuxkit build --no-sbom --format tar --o "${NAME}-1.tar" ./test1.yml
linuxkit build -v --no-sbom --format tar --input-tar "${NAME}-1.tar" --o "${NAME}-2.tar" ./test2.yml 2>&1 | tee ${logfile}

# the logfile should indicate which parts were copied and which not
# we only know this because we built the test2.yml manually

# should have 3 entries copied from init, but not a 4th
errors=""
grep -q "Copy init\[0\]" ${logfile} || errors="${errors}\nmissing Copy init[0]"
grep -q "Copy init\[1\]" ${logfile} || errors="${errors}\nmissing Copy init[1]"
grep -q "Copy init\[2\]" ${logfile} || errors="${errors}\nmissing Copy init[2]"
grep -q "Copy init\[3\]" ${logfile} && errors="${errors}\nunexpected Copy init[3]"
# should have one entry copied from onboot, but not a second
grep -q "Copy onboot\[0\]" ${logfile} || errors="${errors}\nmissing Copy onboot[0]"
grep -q "Copy onboot\[1\]" ${logfile} && errors="${errors}\nunexpected Copy onboot[1]"
# should have one entry copied from services, but not a second or third
grep -q "Copy services\[0\]" ${logfile} || errors="${errors}\nmissing Copy services[0]"
grep -q "Copy services\[1\]" ${logfile} && errors="${errors}\nunexpected Copy services[1]"
grep -q "Copy services\[2\]" ${logfile} && errors="${errors}\nunexpected Copy services[2]"

if [ -n "${errors}" ]; then
	echo "Errors: ${errors}"
	echo "logfile: ${logfile}"
	exit 1
fi

exit 0
