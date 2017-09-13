#!/bin/sh
mount --bind /opt/cni /var/lib/cni/opt
until kubelet --kubeconfig=/var/lib/kubeadm/kubelet.conf \
	      --require-kubeconfig=true \
	      --pod-manifest-path=/var/lib/kubeadm/manifests \
	      --allow-privileged=true \
	      --cluster-dns=10.96.0.10 \
	      --cluster-domain=cluster.local \
	      --cgroups-per-qos=false \
	      --enforce-node-allocatable= \
	      --network-plugin=cni \
	      --cni-conf-dir=/var/lib/cni/etc/net.d \
	      --cni-bin-dir=/var/lib/cni/opt/bin \
	      $@; do
    if [ ! -f /var/config/userdata ] ; then
	sleep 1
    else
	kubeadm join --skip-preflight-checks $(cat /var/config/userdata)
    fi
done
