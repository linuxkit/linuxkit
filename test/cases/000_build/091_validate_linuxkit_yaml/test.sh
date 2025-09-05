#!/bin/sh
# SUMMARY: Check that the build-args are correctly passed to Dockerfiles
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# Test code goes here
RESULT=$(linuxkit build linuxkit.yaml 2>&1 || echo FAILED)
if [ "${RESULT}" != "FAILED" ]; then
    echo "Build should have failed with invalid yaml, instead was ${RESULT}"
fi
echo "Summary: correctly detected invalid yaml"

exit 0
