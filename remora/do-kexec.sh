#!/bin/sh

set -ex

APPEND="$*"

[ -z "$*" ] && APPEND="$(cat /proc/cmdline)"

gzip initrd.img

mv initrd.img.gz vmlinuz64 /var/tmp

kexec -f /var/tmp/vmlinuz64 --initrd=/var/tmp/initrd.img.gz --append="$APPEND"
kexec -e
