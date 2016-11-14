#!/bin/sh

set -e

mobydir=/Applications/Docker.app/Contents/Resources/moby

backup_once() {
	if ! [ -e "$1"- ]
	then
		cp "$1" "$1"-
	fi
}

if [ "$1" = "undo" ]
then
	cp "$mobydir"/initrd.img- "$mobydir"/initrd.img
	cp "$mobydir"/vmlinuz64- "$mobydir"/vmlinuz64
else
	backup_once "$mobydir"/initrd.img
	backup_once "$mobydir"/vmlinuz64
	cp alpine/initrd.img "$mobydir"/initrd.img
	cp alpine/kernel/x86_64/vmlinuz64 "$mobydir"/vmlinuz64
fi

docker run --privileged --pid=host justincormack/nsenter1 /sbin/reboot
