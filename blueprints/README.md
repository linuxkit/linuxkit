# Blueprints

This directory will contain the blueprints for base systems on the platforms that we support with LinuxKit.

These will be used for running tests, and for the low level parts of blueprints for higher level systems.

These include all the platforms that Docker has editions on, and all platforms that our community supports.
The detailed blueprints will be addded soon for at least the following platforms. There are WIP versions in
the [examples/](../examples/) directory.

- MacOS
- Windows Hyper-V
- VMWare
- KVM
- AWS
- Azure
- GCP
- BlueMix
- Packet.net
- ...

### Docker for Mac

An initial blueprint for the open source components of Docker for Mac is available in [docker-for-mac.yml](docker-for-mac.yml). The blueprint has support for controlling `dockerd` from the host via `vsudd` and port forwarding with VPNKit. It requires HyperKit, VPNKit and a Docker client on the host to run. The easiest way to install these at the moment is to install a recent version of Docker for Mac.

To run the VM with a 500M disk:

```
linuxkit run hyperkit -networking=vpnkit -vsock-ports=2375 -disk size=500M docker-for-mac
```

In another terminal you should now be able to access docker via the socket `guest.00000947` in the state directory (`docker-for-mac-state/` by default):

```
$ docker -H unix://docker-for-mac-state/guest.00000947 ps
CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
```

