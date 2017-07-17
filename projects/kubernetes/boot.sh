#!/bin/bash -eu
: ${KUBE_PORT_BASE:=2222}
if [ $# -eq 0 ] ; then
    img="kube-master"
    port=${KUBE_PORT_BASE}
    data=""
    state="kube-master-state"
elif [ $# -gt 1 ] ; then
    case $1 in
	''|*[!0-9]*)
	    echo "Node number must be a number"
	    exit 1
	    ;;
	0)
	    echo "Node number must be greater than 0"
	    exit 1
	    ;;
	*) ;;
    esac
    img="kube-node"
    name="node-${1}"
    port=$((${KUBE_PORT_BASE} + $1))
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
../../bin/linuxkit run -publish $port:22 -cpus 2 -mem 4096 -state "${state}" -disk size=4G -data "${data}" "${img}"
