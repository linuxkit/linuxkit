#!/bin/sh

set -e

cp alpine/initrd.img /Applications/Docker.app/Contents/Resources/moby/initrd.img
cp alpine/kernel/vmlinuz64 /Applications/Docker.app/Contents/Resources/moby/vmlinuz64

killall com.docker.driver.amd64-linux

sleep 2

time docker ps
