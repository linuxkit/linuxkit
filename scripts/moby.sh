#!/bin/sh

set -e

if [ "x$1" == "xundo" ]
then
	cp /Applications/Docker.app/Contents/Resources/moby/initrd.img- /Applications/Docker.app/Contents/Resources/moby/initrd.img
	cp /Applications/Docker.app/Contents/Resources/moby/vmlinuz64- /Applications/Docker.app/Contents/Resources/moby/vmlinuz64
else
	cp /Applications/Docker.app/Contents/Resources/moby/initrd.img /Applications/Docker.app/Contents/Resources/moby/initrd.img-
	cp /Applications/Docker.app/Contents/Resources/moby/vmlinuz64 /Applications/Docker.app/Contents/Resources/moby/vmlinuz64-
	cp alpine/initrd.img /Applications/Docker.app/Contents/Resources/moby/initrd.img
	cp alpine/kernel/vmlinuz64 /Applications/Docker.app/Contents/Resources/moby/vmlinuz64
fi

killall com.docker.driver.amd64-linux

sleep 20

time docker ps
