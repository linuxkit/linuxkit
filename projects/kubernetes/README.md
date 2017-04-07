# Kubernetes

This project aims to demonstrate how one can create minimal and immutable Kubernetes OS images with Moby.

Make sure to `cd projects/kubernetes` first.

Build container & OS images:
```
make
```

Boot Kubernetes master OS image using `hyperkit` on macOS:
```
../../bin/moby run hyperkit -mem 4096 -cpus 2 kube-master
```

Manually initialise master with `kubeadm`:
```
runc exec kubelet kubeadm init --skip-preflight-checks
```

Once `kubeadm` exits, try `runc exec kubelet kubectl get nodes`.
