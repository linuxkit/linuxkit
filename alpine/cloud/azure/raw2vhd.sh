#!/bin/sh

set -e

>&2 echo "Converting raw image file to VHD..."
qemu-img convert -f raw -O vpc -o subformat=fixed /tmp/mobylinux.img /tmp/mobylinux.vhd 1>&2
>&2 echo "Done converting to VHD."
