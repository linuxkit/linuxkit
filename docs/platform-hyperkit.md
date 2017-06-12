# LinuxKit with HyperKit (macOS)

We recommend using LinuxKit in conjunction with
[Docker for Mac](https://docs.docker.com/docker-for-mac/install/). For
the time being it's best to be on the latest edge release. `linuxkit
run` uses [HyperKit](https://github.com/moby/hyperkit) and
[VPNKit](https://github.com/moby/vpnkit) and the edge release ships
with updated versions of both.

Alternatively, you can install HyperKit and VPNKit standalone and use it without Docker for Mac.


## Boot

The HyperKit backend currently only supports booting the
`kernel+initrd` output from `moby` (technically we could support EFI
boot as well).


## Console

With `linuxkit run` on HyperKit the serial console is redirected to
stdio, providing interactive access to the VM. The output of the VM
can be re-directed to a file or pipe, but then stdin is not available.
HyperKit does not provide a console device.


## Disks

The HyperKit backend support configuring a persistent disk using the
standard `linuxkit` `-disk` syntax.  Currently, only one disk is
supported and the disk is in raw format.


## Networking

By default, `linuxkit run` creates a VM with a single network
interface which, logically, is attached to a L2 bridge. The bridge
also has the VM used by Docker for Mac attached to it. This means that
the LinuxKit VMs, created with `linuxkit run`, can be accessed from
containers running on Docker for Mac.

The LinuxKit VMs have IP addresses on the `192.168.65.0/24` subnet
assigned by a DHCP server part of VPNKit. `192.168.65.1` is reserved
for VPNKit as the default gateway and `192.168.65.2` is used by the
Docker for Mac VM.

By default, LinuxKit VMs get incrementally increasing IP addresses,
but you can assign a fixed IP address with `linuxkit run -ip`. It's
best to choose an IP address from the DHCP address range above, but
care must be taken to avoid clashes of IP address.

*NOTE:* The LinuxKit VMs can *not* be directly accessed by IP address
from the host.  Enabling this would require use of the macOS `vmnet`
framework, which requires the VMs to run as `root`.  We don't consider
this option palatable, and provide alternative options to access the
VMs over the network below.


### Accessing services

The simplest way to access networking services exposed by a LinuxKit VM is to use a Docker for Mac container.

For example, to access an ssh server in a LinuxKit VM, create a ssh client container from:
```
FROM alpine:edge
RUN apk add --no-cache openssh-client
```
and then run
```
docker build -t ssh .
docker run --rm -ti -v ~/.ssh:/root/.ssh  ssh ssh <IP address of VM>
```

### Forwarding ports to the host

Ports can be forwarded to the host using a container with `socat` or with VPNKit which comes with Docker for Mac.

#### Port forwarding with `socat`
A `socat` container can be used to proxy between the LinuxKit VM's ports and
localhost.  For example, to expose the redis port from the [RedisOS
example](../examples/redis-os.yml), use this Dockerfile:
```
FROM alpine:edge
RUN apk add --no-cache socat
ENTRYPOINT [ "/usr/bin/socat" ]
```
and then:
```
docker build -t socat .
docker run --rm -t -d -p 6379:6379 socat tcp-listen:6379,reuseaddr,fork tcp:<IP address of VM>:6379
```

#### Port forwarding with VPNKit

VPNKit has the general tooling to expose any guest VM port on the host (just
like it does with containers in Docker for Mac). To enable forwarding, a
`vpnkit-forwarder` container must be running in the VM. The VM also has to be
booted with `linuxkit run hyperkit -networking=vpnkit`.

VPNKit uses a 9P mount in `/port` for coordination between the components.
Port forwarding can be manually set up by creating new directories in `/port`
or by using the `vpnkit-expose-port` tool. More details about the forwarding
mechanism is available in the [VPNKit
documentation](https://github.com/moby/vpnkit/blob/master/docs/ports.md#signalling-from-the-vm-to-the-host).

To get started, the easiest solution at the moment is to use the
`vpnkit-expose-port` command to tell the forwarder and `vpnkit` which ports to
forward. This process requires fewer privileges than `vpnkit-forwarder` and can
be run in a container without networking.

A full example with `vpnkit` forwarding of `sshd` is available in [examples/vpnkit-forwarder.yml](/examples/vpnkit-forwarder.yml).

After building and running the example you should be able to connect to ssh on port 22 on
localhost. The port can also be exposed externally by changing the host IP in
the example to 0.0.0.0.

## Integration services and Metadata

There are no special integration services available for HyperKit, but
there are a number of packages, such as `vsudd`, which enable
tighter integration of the VM with the host (see below).

The HyperKit backend also allows passing custom userdata into the
[metadata pacakge](./metadata.md) using the `-data` command-line
option.


### `vsudd` unix domain socket forwarding

The [`vsudd` package](/pkg/vsudd) provides a daemon that exposes unix
domain socket inside the VM to the host via virtio or Hyper-V sockets.
With HyperKit, the virtio sockets can be exposed as unix domain
sockets on the host, enabling access to other daemons, like
`containerd` and `dockerd`, from the host.  An example configuration
file is available in [examples/vsudd.yml](/examples/vsudd.yml).

After building the example, run it with `linuxkit run hyperkit
-vsock-ports 2374 vsudd`. This will create a unix domain socket in the state directory that maps to the `containerd` control socket. The socket is called `guest.00000946`.

If you install the `ctr` tool on the host you should be able to access the
`containerd` running in the VM:

```
$ go get -u -ldflags -s github.com/containerd/containerd/cmd/ctr
...
$ ctr -a vsudd-state/guest.00000946 list
ID        IMAGE     PID       STATUS
vsudd               466       RUNNING
```
