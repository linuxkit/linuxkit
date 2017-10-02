#!/bin/sh
# Kubelet outputs only to stderr, so arrange for everything we do to go there too
exec 1>&2

if [ ! -e /var/lib/cni/.opt.defaults-extracted ] ; then
    mkdir -p /var/lib/cni/opt/bin
    tar -xzf /root/cni.tgz -C /var/lib/cni/opt/bin
    touch /var/lib/cni/.opt.defaults-extracted
fi
if [ -e /etc/kubelet.sh.conf ] ; then
    . /etc/kubelet.sh.conf
fi

conf=/var/lib/kubeadm/kubelet.conf

if [ -f "${conf}" ] ; then
    echo "kubelet.sh: kubelet already configured"
elif [ -e /var/config/kubeadm/init ] ; then
    echo "kubelet.sh: init cluster with metadata \"$(cat /var/config/kubeadm/init)\""
    # This needs to be in the background since it waits for kubelet to start.
    # We skip printing the token so it is not persisted in the log.
    kubeadm-init.sh --skip-token-print $(cat /var/config/kubeadm/init) &
elif [ -e /var/config/kubeadm/join ] ; then
    echo "kubelet.sh: joining cluster with metadata \"$(cat /var/config/kubeadm/join)\""
    kubeadm join --skip-preflight-checks $(cat /var/config/kubeadm/join)
elif [ -e /var/config/userdata ] ; then
    echo "kubelet.sh: joining cluster with metadata \"$(cat /var/config/userdata)\""
    kubeadm join --skip-preflight-checks $(cat /var/config/userdata)
fi

echo "kubelet.sh: waiting for ${conf}"
# TODO(ijc) is there a race between kubeadm creating this file and
# finishing the write where we might be able to fall through and
# start kubelet with an incomplete configuration file? I've tried
# to provoke such a race without success. An explicit
# synchronisation barrier or changing kubeadm to write
# kubelet.conf atomically might be good in any case.
until [ -f "${conf}" ] ; do
    sleep 1
done

echo "kubelet.sh: ${conf} has arrived" 2>&1

exec kubelet --kubeconfig=${conf} \
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
	      --cadvisor-port=0 \
	      $KUBELET_ARGS $@
