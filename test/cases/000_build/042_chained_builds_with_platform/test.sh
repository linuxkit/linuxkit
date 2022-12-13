#!/bin/sh
# SUMMARY: Check that chained builds get the output of one for the next with --platform defined
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# Test code goes here
echo linuxkit is "$(which linuxkit)"

# build the first, use it to build the second
linuxkit pkg build --force ./build1
linuxkit pkg build --force ./build2

exit 0
