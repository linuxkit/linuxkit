#!/bin/sh
# SUMMARY: Check that all supported output formats are generated
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	# remove any images
	find . -iname "${NAME}*" -exec rm {} \;
}

trap clean_up EXIT

moby build -output tar,kernel+initrd,iso-bios,iso-efi,img-gz,qcow2,vmdk -name "${NAME}" test.yml
[ -f "${NAME}.tar" ] || exit 1
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ] || exit 1
[ -f "${NAME}.iso" ] || exit 1
[ -f "${NAME}-efi.iso" ] || exit 1
[ -f "${NAME}.img.gz" ] || exit 1
[ -f "${NAME}.qcow2" ] || exit 1
# VHD currently requires a lot of memory, disable for now
# [ -f "${NAME}.vhd" ] || exit 1
[ -f "${NAME}.vmdk" ] || exit 1

exit 0
