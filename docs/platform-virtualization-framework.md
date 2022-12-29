# LinuxKit with Virtualization.Framework (macOS)

We recommend using LinuxKit in conjunction with
[Docker for Mac](https://docs.docker.com/docker-for-mac/install/). For
the time being it's best to be on the latest edge release. `linuxkit
run` uses [Virtualization.Framework](https://developer.apple.com/documentation/virtualization) and
[VPNKit](https://github.com/moby/vpnkit) and the edge release ships
with updated versions of both.

Alternatively, you can install Virtualization.Framework and VPNKit standalone and use it without Docker for Mac.

Virtualization.Framework is enabled on macOS only when built with CGO enabled.

## Boot

The Virtualization.Framework backend currently supports booting:
- `kernel+initrd` output from `linuxkit build`.
- `kernel+squashfs` output from `linuxkit build`.
- EFI ISOs using the EFI firmware.

You need to select the boot method manually using the command line
options. The default is `kernel+initrd`. `kernel+squashfs` can be
selected using `-squashfs` and to boot a ISO with EFI you have to
specify `--iso --uefi`.

The `kernel+initrd` uses a RAM disk for the root filesystem. If you
have RAM constraints or large images we recommend using either the
`kernel+squashfs` or the EFI ISO boot.

## Console

With `linuxkit run` on Virtualization.Framework the serial console is redirected to
stdio, providing interactive access to the VM. The output of the VM
can be re-directed to a file or pipe, but then stdin is not available.
Virtualization.Framework does not provide a console device.


## Disks

The Virtualization.Framework backend support configuring a persistent disk using the
standard `linuxkit` `-disk` syntax.  Multiple disks are
supported and the disks are in raw format.

## Power management

Virtualization.Framework sends an ACPI power event when it receives SIGTERM to allow the VM to
shut down properly. The VM has to be able to receive ACPI events to initiate the
shutdown.  This is provided by the [`acpid` package](../pkg/acpid). An example
is available in the [Docker for Mac example](../examples/docker-for-mac.yml).

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


### Accessing network services

Virtualization.Framework offers a number of ways for accessing network services
running inside the LinuxKit VM from the host. These depend on the
networking mode selected via `-networking`. The default mode is
`vmnet`, where it sets up a network bridge. We intend to add support for
`docker-for-mac`, where the same VPNkit instance is shared between
LinuxKit VMs and the VM running as part of Docker for Mac, in the future.

#### Access from the Docker for Mac VM (`-networking docker-for-mac`)

The simplest way to access networking services exposed by a LinuxKit
VM is to use a Docker for Mac container. For example, to access an ssh
server in a LinuxKit VM, create a ssh client container from:

```
FROM alpine:edge
RUN apk add --no-cache openssh-client
```

and then run

```
docker build -t ssh .
docker run --rm -ti -v ~/.ssh:/root/.ssh  ssh ssh <IP address of VM>
```

#### Forwarding ports with `socat`  (`-networking docker-for-mac`)

A `socat` container on Docker for Mac can be used to proxy between the
LinuxKit VM's ports and localhost.  For example, to expose the redis
port from the [RedisOS example](../examples/redis-os.yml), use this
Dockerfile:

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

#### Port forwarding with VPNKit (`-networking docker-for-mac`)

There is **experimental** support for exposing selected ports of the
guest on `localhost` using the `-publish` command line option. For
example, using `-publish 2222:22/tcp` exposes the guest TCP port 22 on
localhost on port 2222. Multiple `-publish` options can be
specified. For example, the image build from the [`sshd
example`](../examples/sshd.yml) can be started with:

```
linuxkit run -publish 2222:22/tcp sshd
```

and then you can log into the LinuxKit VM with `ssh -p 2222
root@localhost`.

Note, this mode is **experimental** and may cause the VPNKit instance
shared with Docker for Mac being confused about which ports are
currently in use, in particular if the LinuxKit VM does not exit
gracefully. This can typically be fixed by restarting Docker for Mac.


#### Port forwarding with VPNKit (`-networking vpnkit`)

An alternative to the previous method is to start your own copy of
`vpnkit` (or connect to an already running instance). This can be done
using the `-networking vpnkit` command line option.

VPNKit uses a 9P mount in `/port` for coordination between
components. The first VM on a VPNKit instance currently needs mount
the 9P filesystem and also needs to run the `vpnkit-forwarder` service
to enable port forwarding to localhost.  A full example with `vpnkit`
forwarding of `sshd` is available in
[examples/vpnkit-forwarder.yml](/examples/vpnkit-forwarder.yml).

To run this example with its own instance of VPNKit, use:

```
linuxkit run -networking vpnkit -publish 2222:22/tcp vpnkit-forwarder
```

You can then access it via:

```
ssh -p 2222 root@localhost
```

More details about the VPNKit forwarding mechanism is available in the
[VPNKit
documentation](https://github.com/moby/vpnkit/blob/master/docs/ports.md#signalling-from-the-vm-to-the-host).


## Integration services and Metadata

There are no special integration services available for Virtualization.Framework, but
there are a number of packages, such as `vsudd`, which enable
tighter integration of the VM with the host (see below).

The Virtualization.Framework backend also allows passing custom userdata into the
[metadata package](./metadata.md) using either the `-data` or `-data-file` command-line
option. This attaches a CD device with the data on.


### `vsudd` unix domain socket forwarding

The [`vsudd` package](/pkg/vsudd) provides a daemon that exposes unix
domain socket inside the VM to the host via virtio or Hyper-V sockets.
With Virtualization.Framework, the virtio sockets can be exposed as unix domain
sockets on the host, enabling access to other daemons, like
`containerd` and `dockerd`, from the host.  An example configuration
file is available in [examples/vsudd-containerd.yml](/examples/vsudd-containerd.yml).

After building the example, run it with `linuxkit run virtualization.framework
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
