#!/bin/sh

set -e

rm -rf /tmp/*

for f in $(ls | grep -vE 'dev|sys|proc|tmp|export|mnt')
do
  cp -a $f /tmp
done

mkdir -m 555 /tmp/dev /tmp/proc /tmp/sys /tmp/mnt
mkdir -m 1777 /tmp/tmp

# these three files are bind mounted in by docker so they are not what we want

cat << EOF > /tmp/etc/hosts
127.0.0.1	localhost
::1	localhost ip6-localhost ip6-loopback
fe00::0	ip6-localnet
ff00::0	ip6-mcastprefix
ff02::1	ip6-allnodes
ff02::2	ip6-allrouters
EOF

cat << EOF > /tmp/etc/resolv.conf
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 2001:4860:4860::8888
nameserver 2001:4860:4860::8844
EOF

printf 'moby' > /tmp/etc/hostname

rm /tmp/mkinitrd.sh

cd /tmp
find . | cpio -H newc -o | gzip -9
