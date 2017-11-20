#!/bin/sh
# SUMMARY: Check that EFI BIOS ISO boots in qemu
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=qemu-efi

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# see https://github.com/linuxkit/linuxkit/issues/1872 this is very flaky in qemu
# disabling for now until we fix a config that works
exit $RT_CANCEL

if command -v qemu-system-x86_64; then
	if [ ! -f /usr/share/ovmf/bios.bin ]; then
		exit $RT_CANCEL
	fi
fi

linuxkit build -format iso-efi -name "${NAME}" test.yml
[ -f "${NAME}-efi.iso" ] || exit 1
linuxkit run qemu -iso -uefi "${NAME}-efi.iso" | grep -q "Welcome to LinuxKit"

exit 0
