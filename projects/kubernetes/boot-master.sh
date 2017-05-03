#!/bin/bash -eu
disk="kube-master-disk.img"
set -x
rm -f "${disk}"
../../bin/linuxkit run -cpus 2 -mem 4096 -disk-size 4096 kube-master
