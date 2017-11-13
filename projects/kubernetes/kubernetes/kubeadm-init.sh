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

if [ -d /var/config/cni/etc/net.d ]; then
  cp /var/config/cni/etc/net.d/* /var/lib/cni/etc/net.d/
fi
# sorting by basename relies on the dirnames having the same number of directories
YAML=$(ls -1 /var/config/kube-system.init/*.yaml /etc/kubeadm/kube-system.init/*.yaml 2>/dev/null | sort --field-separator=/ --key=5)
for i in ${YAML}; do
    n=$(basename "$i")
    if [ -e "$i" ] ; then
	if [ ! -s "$i" ] ; then # ignore zero sized files
	    echo "Ignoring zero size file $n"
	    continue
	fi
	echo "Applying $n"
	if ! kubectl create -n kube-system -f "$i" ; then
	    touch /var/lib/kubeadm/.kubeadm-init.sh-kube-system.init-failed
	    touch /var/lib/kubeadm/.kubeadm-init.sh-kube-system.init-"$n"-failed
	    echo "Failed to apply $n"
	    continue
	fi
    fi
done
if [ -f /var/config/kubeadm/untaint-master ] ; then
    echo "Removing \"node-role.kubernetes.io/master\" taint from all nodes"
    kubectl taint nodes --all node-role.kubernetes.io/master-
fi
touch /var/lib/kubeadm/.kubeadm-init.sh-finished
