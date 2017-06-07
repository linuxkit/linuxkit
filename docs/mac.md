# LinuxKit on a Mac

We recommend using LinuxKit in conjunction with
[Docker for Mac](https://docs.docker.com/docker-for-mac/install/). For
the time being it's best to be on the latest edge release. `linuxkit
run` uses [HyperKit](https://github.com/moby/hyperkit) and
[VPNKit](https://github.com/moby/vpnkit) and the edge release ships
with updated versions of both.


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

While VPNKit has the general tooling to expose any VMs port on the
localhost (just like it does with containers in Docker for Mac), we
are unlikely to expose this as a general feature in `linuxkit run` as
it is very specific to the macOS. However, you can use a `socat` container to proxy between LinuxKit VMs ports and localhost.  For example, to expose the redis port from the [RedisOS example](../examples/redis-os.yml), use this Dockerfile:
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

### Networking Limitations

Due to the VPNKit limitations the `host` is not able to access the `VMs` using its `IPs` (e.g. `$ ssh root@192.168.65.100`)
