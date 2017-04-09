#!/bin/sh

set -x

MOUNTPOINT="$1"

[ -z "$MOUNTPOINT" ] && echo "No mountpoint specified" && exit 1

mkdir -p "$MOUNTPOINT"

mount_drive()
{
	# TODO fix for multiple disks, cdroms etc
	DEVS="$(find /dev -maxdepth 1 -type b ! -name 'loop*' ! -name 'nbd*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort)"

	for DEV in $DEVS
	do
		DRIVE="/dev/${DEV}"

		# see if it has a partition table
		if sfdisk -d "${DRIVE}" >/dev/null 2>/dev/null
		then
			# 83 is Linux partition identifier
			DATA=$(sfdisk -J "$DRIVE" | jq -e -r '.partitiontable.partitions | map(select(.type=="83")) | .[0].node')
			if [ $? -eq 0 ]
			then
				mount "$DATA" "$MOUNTPOINT" && return
			fi
		fi
	done

	echo "WARNING: Failed to mount a persistent volume (is there one?)"
}

mount_drive
