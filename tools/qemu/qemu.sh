#!/bin/sh

set -e

cd /tmp

# extract. BSD tar auto recognises compression, unlike GNU tar
# only if stdin is a tty, if so need files volume mounted...
[ -t 0 ] || bsdtar xzf -

TGZ="$(find . -name '*.tgz' -or -name '*.tar.gz')"
[ -n "$TGZ" ] && bsdtar xzf "$TGZ"

ISO="$(find . -name '*.iso')"
RAW="$(find . -name '*.raw')"
INITRD="$(find . -name '*.img')"
KERNEL="$(find . -name vmlinuz64 -or -name '*bzImage')"

if [ -n "$ISO" ]
then
	ARGS="-cdrom $ISO -drive file=systemdisk.img,format=raw"
elif [ -n "$RAW" ]
then
	# should test with more drives
	ARGS="-drive file=$RAW,format=raw"
elif [ -n "KERNEL" ]
then
	ARGS="-kernel $KERNEL"
	if [ -n "$INITRD" ]
	then
		ARGS="$ARGS -initrd $INITRD"
	fi
	ARGS="$ARGS -append console=ttyS0 -drive file=systemdisk.img,format=raw"
else
	echo "no recognised boot media" >2
	exit 1
fi

echo "$ARGS" | grep -q systemdisk && qemu-img create -f raw systemdisk.img 256M

qemu-system-x86_64 -device virtio-rng-pci -serial stdio -vnc none -m 1024 $ARGS $*
