# Kubernetes

This project aims to demonstrate how one can create minimal and immutable Kubernetes OS images with Moby.

Make sure to `cd projects/kubernetes` first.

Build container & OS images:
```
make
```

Boot Kubernetes master OS image using `hyperkit` on macOS:
```
./boot-master.sh
```

Manually initialise master with `kubeadm`:
```
runc exec kubelet kubeadm-init.sh
```

Once `kubeadm` exits, try `runc exec kubelet kubectl get nodes`.
