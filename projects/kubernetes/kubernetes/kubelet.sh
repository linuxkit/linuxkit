#!/bin/sh
# Kubelet outputs only to stderr, so arrange for everything we do to go there too
exec 1>&2

if [ -e /etc/kubelet.sh.conf ] ; then
    . /etc/kubelet.sh.conf
fi

if [ -f /var/config/kubelet/disabled ] ; then
    echo "kubelet.sh: /var/config/kubelet/disabled file is present, exiting"
    exit 0
fi
if [ -n "$KUBELET_DISABLED" ] ; then
    echo "kubelet.sh: KUBELET_DISABLED environ variable is set, exiting"
    exit 0
fi

if [ ! -e /var/lib/cni/.opt.defaults-extracted ] ; then
    mkdir -p /var/lib/cni/opt/bin
    tar -xzf /root/cni.tgz -C /var/lib/cni/opt/bin
    touch /var/lib/cni/.opt.defaults-extracted
fi

await=/etc/kubernetes/kubelet.conf

if [ -f "/etc/kubernetes/kubelet.conf" ] ; then
    echo "kubelet.sh: kubelet already configured"
elif [ -d /var/config/kubeadm ] ; then
    if [ -f /var/config/kubeadm/init ] ; then
	echo "kubelet.sh: init cluster with metadata \"$(cat /var/config/kubeadm/init)\""
	# This needs to be in the background since it waits for kubelet to start.
	# We skip printing the token so it is not persisted in the log.
	kubeadm-init.sh --skip-token-print $(cat /var/config/kubeadm/init) &
    elif [ -e /var/config/kubeadm/join ] ; then
	echo "kubelet.sh: joining cluster with metadata \"$(cat /var/config/kubeadm/join)\""
	kubeadm join --skip-preflight-checks $(cat /var/config/kubeadm/join)
	await=/etc/kubernetes/bootstrap-kubelet.conf
    fi
elif [ -e /var/config/userdata ] ; then
    echo "kubelet.sh: joining cluster with metadata \"$(cat /var/config/userdata)\""
    kubeadm join --skip-preflight-checks $(cat /var/config/userdata)
    await=/etc/kubernetes/bootstrap-kubelet.conf
fi

echo "kubelet.sh: waiting for ${await}"
# TODO(ijc) is there a race between kubeadm creating this file and
# finishing the write where we might be able to fall through and
# start kubelet with an incomplete configuration file? I've tried
# to provoke such a race without success. An explicit
# synchronisation barrier or changing kubeadm to write
# kubelet.conf atomically might be good in any case.
until [ -f "${await}" ] ; do
    sleep 1
done

echo "kubelet.sh: ${await} has arrived" 2>&1

mkdir -p /etc/kubernetes/manifests

exec kubelet --kubeconfig=/etc/kubernetes/kubelet.conf \
	      --bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
	      --pod-manifest-path=/etc/kubernetes/manifests \
	      --allow-privileged=true \
	      --cluster-dns=10.96.0.10 \
	      --cluster-domain=cluster.local \
	      --cgroups-per-qos=false \
	      --enforce-node-allocatable= \
	      --network-plugin=cni \
	      --cni-conf-dir=/var/lib/cni/etc/net.d \
	      --cni-bin-dir=/var/lib/cni/opt/bin \
	      --cadvisor-port=0 \
	      $KUBELET_ARGS $@
