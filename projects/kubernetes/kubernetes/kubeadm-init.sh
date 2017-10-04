#!/bin/sh
set -e
touch /var/lib/kubeadm/.kubeadm-init.sh-started
if [ -f /etc/kubeadm/kubeadm.yaml ]; then
    echo Using the configuration from /etc/kubeadm/kubeadm.yaml
    if [ $# -ne 0 ] ; then
        echo WARNING: Ignoring command line options: $@
    fi
    kubeadm init --skip-preflight-checks --config /etc/kubeadm/kubeadm.yaml
else
    kubeadm init --skip-preflight-checks --kubernetes-version @KUBERNETES_VERSION@ $@
fi
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
