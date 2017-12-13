#!/bin/sh
# SUMMARY: Check that legacy BIOS ISO boots in qemu
# LABELS: amd64

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-iso

clean_up() {
	rm -rf ${NAME}*
}
trap clean_up EXIT

linuxkit build -format iso-bios -name "${NAME}" test.yml
[ -f "${NAME}.iso" ] || exit 1
linuxkit run qemu -iso "${NAME}.iso" | grep -q "Welcome to LinuxKit"

exit 0
