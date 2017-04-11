#!/bin/bash -eux
rm -f kube-master-disk.img
../../bin/moby run hyperkit -cpus 2 -mem 4096 -disk-size 2048 kube-master
