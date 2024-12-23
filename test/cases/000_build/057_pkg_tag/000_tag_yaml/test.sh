#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

linuxkit pkg build --force .

# just run docker image inspect; if it does not exist, it will error out
linuxkit cache ls 2>&1 | grep 'linuxkit/image-with-tag:file'

exit 0
