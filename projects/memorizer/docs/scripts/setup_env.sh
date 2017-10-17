#!/bin/sh

mount -t debugfs nodev /sys/kernel/debug
mknod /dev/null c 1 3
cp cp_test.sh /root
cp userApp /root
cp enable_memorizer.sh /root
cp disable_memorizer.sh /root
cd /root && mknod node c 252 0


