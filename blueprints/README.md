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

An initial blueprint for the open source components of Docker for Mac is available in [docker-for-mac](docker-for-mac). The blueprint has support for controlling `dockerd` from the host via `vsudd` and port forwarding with VPNKit. It requires HyperKit, VPNKit and a Docker client on the host to run. The easiest way to install these at the moment is to install a recent version of Docker for Mac.

To build it with the latest Docker CE:

```
$ moby build -name docker-for-mac base.yml docker-ce.yml
```

To run the VM with a 500M disk:

```
linuxkit run hyperkit -networking=vpnkit -vsock-ports=2376 -disk size=500M -data ./metadata.json docker-for-mac
```

In another terminal you should now be able to access docker via the socket `guest.00000947` in the state directory (`docker-for-mac-state/` by default):

```
$ docker -H unix://docker-for-mac-state/guest.00000948 ps
CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
```

### Linux Containers On Windows (LCOW)

The [LCOW](./lcow.yml) file contains the blueprint for building a
minimal Linux kernel and initrd for Linux Containers on
Windows. Invoke it with `moby build lcow.yml` and you get a
`lcow-kernel` and `lcow-initrd.img`. Rename `lcow-kernel` to
`bootx64.efi` and `lcow-initrd.img` to `initrd.img` and then
follow
[these instructions](https://github.com/moby/moby/issues/33850). The
process for creating the image is
documented [here](https://github.com/Microsoft/opengcs).
