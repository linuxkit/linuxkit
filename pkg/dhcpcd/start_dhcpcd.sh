#!bin/sh

set -ex

ip link set eth0 up

cp /dhcpcd.conf /etc/dhcpcd.conf

/sbin/dhcpcd --nobackground