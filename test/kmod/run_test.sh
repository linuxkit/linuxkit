#! /bin/sh

# Make sure we have the latest kernel image
docker pull linuxkit/kernel:4.9.x
# Build a package
docker build -t kmod-test .
# Build a LinuxKit image with kernel module (and test script)
moby build kmod
# Run it
linuxkit run kmod
