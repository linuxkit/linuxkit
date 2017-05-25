#!/bin/sh
# SUMMARY: Test the mkimage container by using it to build a bootable qcow2
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -iname "run*" -not -iname "*.yml" -exec rm -rf {} \;
	find . -iname "mkimage*" -not -iname "*.yml" -exec rm -rf {} \;
	rm -f disk.qcow2
}
trap clean_up EXIT

# Test code goes here
moby build run.yml
moby build mkimage.yml
linuxkit run qemu -disk-size 200 -disk-format qcow2 -disk disk.qcow2 -kernel mkimage
linuxkit run qemu disk.qcow2

exit 0
