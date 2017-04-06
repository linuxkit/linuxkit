#!/bin/sh

# this script assumes anything on the disk can be removed if corrupted
# other use cases may need different scripts.

# currently only supports ext4 but should be expanded

do_fsck()
{
	# preen
	/sbin/e2fsck -p $*
	EXIT_CODE=$?
	# exit code 1 is errors corrected
	[ "${EXIT_CODE}" -eq 1 ] && EXIT_CODE=0
	# exit code 2 or 3 means need to reboot
	[ "${EXIT_CODE}" -eq 2 -o "${EXIT_CODE}" -eq 3 ] && /sbin/reboot
	# exit code 4 or over is fatal
	[ "${EXIT_CODE}" -lt 4 ] && return "${EXIT_CODE}"

	# try harder
	/sbin/e2fsck -y $*
	# exit code 1 is errors corrected
	[ "${EXIT_CODE}" -eq 1 ] && EXIT_CODE=0
	# exit code 2 or 3 means need to reboot
	[ "${EXIT_CODE}" -eq 2 -o "${EXIT_CODE}" -eq 3 ] && /sbin/reboot
	# exit code 4 or over is fatal
	[ "${EXIT_CODE}" -ge 4 ] && printf "Filesystem unrecoverably corrupted, will reformat\n"

	return "${EXIT_CODE}"
}

do_fsck_extend_mount()
{
	DRIVE="$1"
	DATA="$2"

	do_fsck "$DATA" || return 1

	# only try to extend if there is a single partition and free space
	PARTITIONS=$(sfdisk -J "$DRIVE" | jq '.partitiontable.partitions | length')

	if [ "$PARTITIONS" -eq 1 ] && \
		sfdisk -F "$DRIVE" | grep -q 'Unpartitioned space' &&
		! sfdisk -F "$DRIVE" | grep -q '0 B, 0 bytes, 0 sectors'
	then
		SPACE=$(sfdisk -F "$DRIVE" | grep 'Unpartitioned space')
		printf "Resizing disk partition: $SPACE\n"

		# 83 is Linux partition id
		START=$(sfdisk -J "$DRIVE" | jq -e '.partitiontable.partitions | map(select(.type=="83")) | .[0].start')

		sfdisk -q --delete "$DRIVE" 2> /dev/null
		echo "${START},,83;" | sfdisk -q "$DRIVE"

		# set bootable flag
		sfdisk -A "$DRIVE" 1

		# update status
		blockdev --rereadpt $diskdev 2> /dev/null
		mdev -s

		# wait for device
		for i in $(seq 1 50); do test -b "$DATA" && break || sleep .1; mdev -s; done

		# resize2fs fails unless we use -f here
		do_fsck -f "$DATA" || return 1
		resize2fs "$DATA"

		do_fsck "$DATA" || return 1
	fi
}

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

# TODO fix for multiple disks, cdroms etc
DEV="$(find /dev -maxdepth 1 -type b ! -name 'loop*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort | head -1 )"

[ -z "${DEV}" ] && exit 1

DRIVE="/dev/${DEV}"

# see if it has a partition table already
if sfdisk -d "${DRIVE}" >/dev/null 2>/dev/null
then
	DATA=$(sfdisk -J "$DRIVE" | jq -e -r '.partitiontable.partitions | map(select(.type=="83")) | .[0].node')
	if [ $? -eq 0 ]
	then
		do_fsck_extend_mount "$DRIVE" "$DATA" || do_mkfs "$DRIVE"
	else
		do_mkfs "$DRIVE"
	fi
else
	do_mkfs "$DRIVE"
fi
