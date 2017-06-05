#!/bin/sh

# vpnkit mount point for port forwarding
echo "Mounting /dfm/port"
mount -v -t 9p -o trans=virtio,dfltuid=1001,dfltgid=50,version=9p2000 port /dfm/port

# copy plugins if new disk
echo "Configuring /var/lib/docker"
[ ! -d /var/lib/docker/plugins ] && cp -va /dfm/docker/* /var/lib/docker/
