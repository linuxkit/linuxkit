#!/bin/sh
# SUMMARY: Check that qcow2 image boots in qemu
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-qcow2

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${NAME}* || true
}

trap clean_up EXIT

moby build -output qcow2 -name "${NAME}" test.yml
[ -f "${NAME}.qcow2" ] || exit 1
linuxkit run qemu "${NAME}.qcow2" | grep -q "Welcome to LinuxKit"
exit 0
