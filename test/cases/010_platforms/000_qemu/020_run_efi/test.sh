#!/bin/sh
# SUMMARY: Check that legacy BIOS ISO boots in qemu
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-efi

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${NAME}* || true
}

trap clean_up EXIT

if command -v qemu-system-x86_64; then
	if [ ! -f /usr/share/ovmf/bios.bin ]; then
		exit $RT_CANCEL
	fi
fi

moby build -output iso-efi -name "${NAME}" test.yml
[ -f "${NAME}-efi.iso" ] || exit 1
linuxkit run qemu -iso -uefi "${NAME}-efi.iso" | grep -q "Welcome to LinuxKit"
exit 0
