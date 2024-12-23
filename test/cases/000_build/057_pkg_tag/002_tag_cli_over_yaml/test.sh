#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

linuxkit pkg build --force --tag cli .

# just run docker image inspect; if it does not exist, it will error out
linuxkit cache ls 2>&1 | grep 'linuxkit/image-with-tag:cli'

# specifically, the `file` tag should not exist, so check that it does not exist
if linuxkit cache ls 2>&1 | grep 'linuxkit/image-with-tag:file-new'; then
    echo "ERROR: image with tag 'file-new' should not exist"
    exit 1
fi

exit 0
