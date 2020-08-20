#!/bin/sh
set -e
cat << "EOF" | docker run -i --rm -v $PWD:/data alpine:3.8
set -e
apk --update add xorriso
mkdir -p /build
cd /build
mkdir foo bar abc
dd if=/dev/zero of=bar/largefile bs=1M count=5
dd if=/dev/zero of=abc/largefile bs=1M count=5
i=0
until [ $i -gt 75 ]; do echo "filename_"${i} > foo/filename_${i}; i=$(( $i+1 )); done
ln -s /a/b/c/d/ef/g/h link
deepdir="deep/a/b/c/d/e/f/g/h/i/j/k"
mkdir -p ${deepdir}
echo file > ${deepdir}/file
echo README > README.md
# we will generate two isos: one with Rock Ridge, one with pure iso9660
# be explicit about what we support - we want iso9660 compliance "deep_paths_off" so we can test it
xorriso -rockridge off -as mkisofs -o /data/9660.iso .
# be explicit about what we support - we want iso9660 compliance "deep_paths_off" so we can test it
xorriso -compliance "clear:only_iso_version:deep_paths_off:long_paths:no_j_force_dots:always_gmt:old_rr" -as mkisofs -o /data/rockridge.iso .
EOF
