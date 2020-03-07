# open-vm-tools
This should allow end-users to gracefully reboot or shutdown Kubernetes nodes (incuding control planes) running on vSphere Hypervisor.

Furthermore, it is also mandatory to have `open-vm-tools` installed on your Kubernetes nodes to use vSphere Cloud Provider (i.e. determinte virtual machine's FQDN).

## Remarks:
- `spec.template.spec.hostNetwork: true`: correctly report node IP address; required
- `spec.template.spec.hostPID: true`: send the right signal to node, instead of killing the container itself; required
- `spec.template.spec.priorityClassName: system-cluster-critical`: critical to a fully functional cluster
- `spec.template.spec.securityContext.privileged: true`: gain more privileges than its parent process; required
