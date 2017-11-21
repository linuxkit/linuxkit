#!/bin/sh
# SUMMARY: Test the node_exporter example
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=node_exporter

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

# Test code goes here
linuxkit build "${LINUXKIT_EXAMPLES_DIR}/${NAME}.yml" 

exit 0

