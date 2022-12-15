#!/bin/sh
# SUMMARY: Check that raw image boots in qemu
# LABELS: amd64

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-raw

clean_up() {
	rm -rf ${NAME}*
}
trap clean_up EXIT

linuxkit build --format raw-bios --name "${NAME}" test.yml
[ -f "${NAME}-bios.img" ] || exit 1
linuxkit run qemu "${NAME}-bios.img" | grep -q "Welcome to LinuxKit"

exit 0
