#!/bin/sh
set -e
touch /var/lib/kubeadm/.kubeadm-init.sh-started
kubeadm init --skip-preflight-checks --kubernetes-version @KUBERNETES_VERSION@ $@
for i in /etc/kubeadm/kube-system.init/*.yaml ; do
    if [ -e "$i" ] ; then
	echo "Applying "$(basename "$i")
	kubectl create -n kube-system -f "$i"
    fi
done
if [ -f /var/config/kubeadm/untaint-master ] ; then
    echo "Removing \"node-role.kubernetes.io/master\" taint from all nodes"
    kubectl taint nodes --all node-role.kubernetes.io/master-
fi
touch /var/lib/kubeadm/.kubeadm-init.sh-finished
