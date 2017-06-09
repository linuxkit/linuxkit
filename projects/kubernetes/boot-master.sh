#!/bin/bash -eu
disk="kube-master-disk.img"
set -x
rm -f "${disk}"
../../bin/linuxkit run -cpus 2 -mem 4096 -disk "${disk}",size=4G kube-master
