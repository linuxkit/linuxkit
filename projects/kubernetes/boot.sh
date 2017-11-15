#!/bin/sh

set -e

: ${KUBE_MASTER_VCPUS:=2}
: ${KUBE_MASTER_MEM:=1024}
: ${KUBE_MASTER_DISK:=4G}
: ${KUBE_MASTER_UNTAINT:=n}

: ${KUBE_NODE_VCPUS:=2}
: ${KUBE_NODE_MEM:=4096}
: ${KUBE_NODE_DISK:=8G}

: ${KUBE_NETWORKING:=default}
: ${KUBE_RUN_ARGS:=}
: ${KUBE_EFI:=}
: ${KUBE_MAC:=}
: ${KUBE_CLEAR_STATE:=}

[ "$(uname -s)" = "Darwin" ] && KUBE_EFI=1

suffix=".iso"
[ -n "${KUBE_EFI}" ] && suffix="-efi.iso" && uefi="--uefi"

if [ $# -eq 0 ] ; then
    img="kube-master"
    # If $KUBE_MASTER_AUTOINIT is set, including if it is set to ""
    # then we configure for auto init. If it is completely unset then
    # we do not.
    if [ -n "${KUBE_MASTER_AUTOINIT+x}" ] ; then
	kubeadm_data="${kubeadm_data+$kubeadm_data, }\"init\": { \"content\": \"${KUBE_MASTER_AUTOINIT}\" }"
    fi
    if [ "${KUBE_MASTER_UNTAINT}" = "y" ] ; then
	kubeadm_data="${kubeadm_data+$kubeadm_data, }\"untaint-master\": { \"content\": \"\" }"
    fi

    state="kube-master-state"

    : ${KUBE_VCPUS:=$KUBE_MASTER_VCPUS}
    : ${KUBE_MEM:=$KUBE_MASTER_MEM}
    : ${KUBE_DISK:=$KUBE_MASTER_DISK}
elif [ $# -ge 1 ] ; then
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
    shift

    if [ $# -ge 1 ] ; then
	kubeadm_data="\"join\": { \"content\": \"${*}\" }"
    fi

    state="kube-${name}-state"

    : ${KUBE_VCPUS:=$KUBE_NODE_VCPUS}
    : ${KUBE_MEM:=$KUBE_NODE_MEM}
    : ${KUBE_DISK:=$KUBE_NODE_DISK}
else
    echo "Usage:"
    echo " - Boot master:"
    echo "   ${0}"
    echo " - Boot node:"
    echo "   ${0} <node> <join_args>"
    exit 1
fi

if [ -n "${kubeadm_data}" ] ; then
    data="{  \"kubeadm\": { \"entries\": { ${kubeadm_data} } } }"
fi

set -x
if [ -n "${KUBE_CLEAR_STATE}" ] ; then
    rm -rf "${state}"
    mkdir "${state}"
    if [ -n "${KUBE_MAC}" ] ; then
	echo -n "${KUBE_MAC}" > "${state}"/mac-addr
    fi
fi
linuxkit run ${KUBE_RUN_ARGS} -networking ${KUBE_NETWORKING} -cpus ${KUBE_VCPUS} -mem ${KUBE_MEM} -state "${state}" -disk size=${KUBE_DISK} -data "${data}" ${uefi} "${img}${suffix}"
