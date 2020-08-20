#!/bin/sh
set -e
set +x

cat <<"EOF" | docker run -i --rm -v $PWD:/data alpine:3.8 
set -e
set +x
apk --update add mtools dosfstools
mkfs.vfat -F 32 -v -C /tmp/boot.img 10000
mmd -i /tmp/boot.img ::/A
mmd -i /tmp/boot.img ::/b
echo testfile > testfile
echo sub > sub
dd if=/dev/random of=large bs=1M count=5
mcopy -i /tmp/boot.img testfile ::/
mcopy -i /tmp/boot.img sub ::/b
mcopy -i /tmp/boot.img large ::/b
cp /tmp/boot.img /data
EOF

