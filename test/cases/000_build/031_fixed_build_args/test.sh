#!/bin/sh
# SUMMARY: Check that the build-args are correctly passed to Dockerfiles
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# Test code goes here
echo Linuxkit is "$(which linuxkit)"
RESULT="$(2>&1 linuxkit pkg build --force . | grep 'Build-arg test' || echo)"
echo RESULT="${RESULT}"
echo "${RESULT}" | grep  -v "FAILED"

exit 0
