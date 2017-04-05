#!bin/sh

set -ex

ip link set eth0 up

cp /dhcpcd.conf /etc/dhcpcd.conf

exec /sbin/dhcpcd --nobackground
