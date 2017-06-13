#!/bin/bash -eu
if [ $# -eq 0 ] ; then
    img="kube-master"
    data=""
    state="kube-master-state"
elif [ $# -gt 1 ] ; then
    img="kube-node"
    name="node-${1}"
    shift
    data="${*}"
    state="kube-${name}-state"
else
    echo "Usage:"
    echo " - Boot master:"
    echo "   ${0}"
    echo " - Boot node:"
    echo "   ${0} <node> <join_args>"
    exit 1
fi
set -x
rm -rf "${state}"
../../bin/linuxkit run -cpus 2 -mem 4096 -state "${state}" -disk size=4G -data "${data}" "${img}"
