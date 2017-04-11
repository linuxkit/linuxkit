# Kubernetes

This project aims to demonstrate how one can create minimal and immutable Kubernetes OS images with Moby.

Make sure to `cd projects/kubernetes` first.

Edit `kube-master.yml` and add your public SSH key to `files` section.

Build OS images:
```
make build-vm-images
```

Boot Kubernetes master OS image using `hyperkit` on macOS:
```
./boot-master.sh
```

Get IP address of the master:
```
ip addr show dev eth0
```

Login to the kubelet container:
```
./ssh_into_kubelet.sh <master-ip>
```

Manually initialise master with `kubeadm`:
```
kubeadm-init.sh
```

Once `kubeadm` exits, make sure to copy the `kubeadm join` arguments,
and try `runc exec kubelet kubectl get nodes`.

To boot a node use:
```
./boot-node.sh <n> [<join_args> ...]
```

More specifically, to start 3 nodes use 3 separate shells and run this:
```
shell1> ./boot-node.sh 1 --token bb38c6.117e66eabbbce07d 192.168.65.22:6443
shell2> ./boot-node.sh 2 --token bb38c6.117e66eabbbce07d 192.168.65.22:6443
shell3> ./boot-node.sh 3 --token bb38c6.117e66eabbbce07d 192.168.65.22:6443
```
