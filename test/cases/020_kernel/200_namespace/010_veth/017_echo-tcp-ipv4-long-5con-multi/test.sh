#!/bin/sh
# SUMMARY: Namespace stress with multiple instances of 5 concurrent long running TCP/IPv4 connections over a veth pair
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-ns

clean_up() {
    rm -rf ${NAME}-*
}
trap clean_up EXIT

linuxkit build --format kernel+initrd --name ${NAME} ../../common.yml test.yml
RESULT="$(linuxkit run -cpus 2 ${NAME})"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
