#!/bin/bash -eu
if [ $# -eq 0 ] ; then
    img="kube-master"
    disk="kube-master-disk.img"
    data=""
elif [ $# -gt 1 ] ; then
    img="kube-node"
    name="node-${1}"
    shift
    disk="kube-${name}-disk.img"
    data="${*}"
else
    echo "Usage:"
    echo " - Boot master:"
    echo "   ${0}"
    echo " - Boot node:"
    echo "   ${0} <node> <join_args>"
    exit 1
fi
set -x
rm -f "${disk}"
../../bin/linuxkit run -cpus 2 -mem 4096 -disk "${disk}",size=4G -data "${data}" "${img}"
