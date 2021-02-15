#!/bin/sh
# SUMMARY: Test the mkimage container by using it to build a bootable qcow2
# LABELS: amd64
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	find . -depth -iname "run*" -not -iname "*.yml" -exec rm -rf {} \;
	find . -depth -iname "mkimage*" -not -iname "*.yml" -exec rm -rf {} \;
	find . -depth -iname "disk.qcow2*" -not -iname "*.yml" -exec rm -rf {} \;
	rm -f tarball.img
}
trap clean_up EXIT

# Test code goes here
linuxkit build -format kernel+initrd run.yml
linuxkit build -format kernel+initrd mkimage.yml
tar cf tarball.img run-kernel run-initrd.img run-cmdline
linuxkit run qemu -disk disk.qcow2,size=200M,format=qcow2 -disk tarball.img,format=raw -kernel mkimage
linuxkit run qemu disk.qcow2

exit 0
