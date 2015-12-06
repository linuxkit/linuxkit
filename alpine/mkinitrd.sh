#!/bin/sh

set -e

rm -rf /tmp/*

for f in $(ls | grep -vE 'dev|sys|proc|tmp|export')
do
  cp -a $f /tmp
done

mkdir -m 555 /tmp/dev /tmp/proc /tmp/sys
mkdir -m 1777 /tmp/tmp

cd /tmp/dev

mknod -m 666 null c 1 3
mknod -m 666 full c 1 7
mknod -m 666 ptmx c 5 2
mknod -m 644 random c 1 8
mknod -m 644 urandom c 1 9
mknod -m 666 zero c 1 5
mknod -m 666 tty c 5 0

mknod -m 600 ttyS0 c 4 64

# we are using sata emulation at present
mknod -m 600 sda b 8 0
mknod -m 600 sda1 b 8 1
mknod -m 600 sda2 b 8 2
mknod -m 600 sda3 b 8 3
mknod -m 600 sda4 b 8 4
mknod -m 600 sda5 b 8 5
mknod -m 600 sda6 b 8 6
mknod -m 600 sdb b 8 16
mknod -m 600 sdb1 b 8 17
mknod -m 600 sdb2 b 8 18
mknod -m 600 sdb3 b 8 19
mknod -m 600 sdb4 b 8 20
mknod -m 600 sdb5 b 8 21
mknod -m 600 sdb6 b 8 22

mkdir pty

cd /tmp

find . | cpio -H newc -o > /export/initrd.img
