#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

set +e
linuxkit pkg build --force .
command_status=$?
set -e

if [ $command_status -eq 0 ]; then
    echo "Command should have failed"
    exit 1
fi

exit 0
