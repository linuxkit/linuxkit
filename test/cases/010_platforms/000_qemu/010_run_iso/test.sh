#!/bin/sh
# SUMMARY: Check that legacy BIOS ISO boots in qemu
# LABELS:
# AUTHOR: Dave Tucker <dt@docker.com>
# AUTHOR: Rolf Neugebauer <rolf.neugebauer@docker.com>

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-iso

clean_up() {
	rm -rf ${NAME}* || true
}

trap clean_up EXIT

moby build -name "${NAME}" test.yml
[ -f "${NAME}.iso" ] || exit 1
linuxkit run qemu -iso "${NAME}" | grep -q "Welcome to LinuxKit"
exit 0
