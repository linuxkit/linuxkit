#!/bin/sh

set -e

rm -rf /tmp/*

for f in $(ls | grep -vE 'dev|sys|proc|tmp|export|mnt')
do
  cp -a $f /tmp
done

mkdir -m 555 /tmp/dev /tmp/proc /tmp/sys /tmp/mnt
mkdir -m 1777 /tmp/tmp

cd /tmp/dev

# start from clean slate
rm -rf *

mknod -m 666 null c 1 3
mknod -m 666 full c 1 7
mknod -m 666 ptmx c 5 2
mknod -m 644 random c 1 8
mknod -m 644 urandom c 1 9
mknod -m 666 zero c 1 5
mknod -m 666 tty c 5 0
mknod -m 600 console c 5 1

mknod -m 600 ttyS0 c 4 64
mknod -m 600 fuse c 10 229

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

# mount points in /dev
mkdir pts mqueue shm

# symlinks
ln -s /proc/kcore core
ln -s /proc/self/fd fd
ln -s /proc/self/fd/0 stdin
ln -s /proc/self/fd/1 stdout
ln -s /proc/self/fd/2 stderr

# loop devs
mknod -m 600 loop-control c 10 237
mknod -m 600 loop0 c 7 0
mknod -m 600 loop1 c 7 1
mknod -m 600 loop2 c 7 2
mknod -m 600 loop3 c 7 3
mknod -m 600 loop4 c 7 4
mknod -m 600 loop5 c 7 5
mknod -m 600 loop6 c 7 6
mknod -m 600 loop7 c 7 7

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

printf 'docker' > /tmp/etc/hostname

rm /tmp/bin/mkinitrd.sh

cd /tmp
find . | cpio -H newc -o > /mnt/initrd.img
