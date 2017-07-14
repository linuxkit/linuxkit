#!/bin/sh

# This is a skanky hack. opengcs assume udhcp being preset in the
# rootfs. We don't have it in Alpine's version of busybox so we copy
# this script to /bin/udhcpc and kick dhcpcd in single shot mode.

echo "$@" >> /tmp/dhcpd.log

/sbin/dhcpcd --nobackground -f /etc/dhcpcd.conf -1  >> /tmp/dhcpd.log
