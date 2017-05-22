#!/bin/sh

# currently only supports ext4 disks; will extend for other filesystems, ISO, ...

do_mkfs()
{
	diskdev="$1"

	# new disks does not have an DOS signature in sector 0
	# this makes sfdisk complain. We can workaround this by letting
	# fdisk create that DOS signature, by just do a "w", a write.
	# http://bugs.alpinelinux.org/issues/145
	echo "w" | fdisk $diskdev >/dev/null

	# format one large partition
	echo ";" | sfdisk --quiet $diskdev

	# update status
	blockdev --rereadpt $diskdev 2> /dev/null

	# wait for device
	for i in $(seq 1 50); do test -b "$DATA" && break || sleep .1; mdev -s; done

	FSOPTS="-O resize_inode,has_journal,extent,huge_file,flex_bg,uninit_bg,64bit,dir_nlink,extra_isize"

	mkfs.ext4 -q -F $FSOPTS ${diskdev}1
}

DEV="$(find /dev -maxdepth 1 -type b ! -name 'loop*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort | head -1 )"

[ -z "${DEV}" ] && exit 1

DRIVE="/dev/${DEV}"

# format
do_mkfs "$DRIVE"

PARTITION="${DRIVE}1"

# mount
mount "$PARTITION" /mnt

# copy kernel, initrd
cp -a /data/kernel /data/initrd.img /mnt/

# create syslinux.cfg
CMDLINE="$(cat /data/cmdline)"

CFG="DEFAULT linux
LABEL linux
    KERNEL /kernel
    INITRD /initrd.img
    APPEND ${CMDLINE}
"

printf "$CFG" > /mnt/syslinux.cfg

# install syslinux
extlinux --install /mnt

# unmount
umount /mnt

# install mbr
dd if=usr/share/syslinux/mbr.bin of="$DRIVE" bs=440 count=1

# make bootable
sfdisk -A "$DRIVE" 1
