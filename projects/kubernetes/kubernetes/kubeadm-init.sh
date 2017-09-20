#!/bin/sh
set -e
kubeadm init --skip-preflight-checks --kubernetes-version @KUBERNETES_VERSION@
for i in /etc/kubeadm/kube-system.init/*.yaml ; do
    if [ -e "$i" ] ; then
	echo "Applying "$(basename "$i")
	kubectl create -n kube-system -f "$i"
    fi
done
