# Kubernetes and LinuxKit

This project aims to demonstrate how one can create minimal and immutable Kubernetes OS images with LinuxKit.

Make sure to change directory to `projects/kubernetes` first.

Build OS images:
```
make build-vm-images
```
Please note: The make process copies your public ssh key to the
`kube-master.yml` and `kube-node.yml` files adding your public key to the
authorized_key file so you are able to ssh to systems running these images.

Boot Kubernetes master OS image using the `linuxkit run` command:
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
and try `kubectl get nodes` from within the master.

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
