#!/bin/sh

set -e

mobydir=/Applications/Docker.app/Contents/Resources/moby

if [ "$1" = "undo" ]
then
	cp "$mobydir"/initrd.img- "$mobydir"/initrd.img
	cp "$mobydir"/vmlinuz64- "$mobydir"/vmlinuz64
else
	cp "$mobydir"/initrd.img "$mobydir"/initrd.img-
	cp "$mobydir"/vmlinuz64 "$mobydir"/vmlinuz64-
	cp alpine/initrd.img "$mobydir"/initrd.img
	cp alpine/kernel/x86_64/vmlinuz64 "$mobydir"/vmlinuz64
fi

killall com.docker.driver.amd64-linux

sleep 20

time docker ps
