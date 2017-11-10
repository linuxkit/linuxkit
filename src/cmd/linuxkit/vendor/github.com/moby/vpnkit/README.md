VPN-friendly networking devices for [HyperKit](https://github.com/moby/hyperkit)
===============================

[![Build Status (OSX)](https://circleci.com/gh/moby/vpnkit.png)](https://circleci.com/gh/moby/vpnkit)

Binary artefacts are built by CI:

- [MacOS](https://circleci.com/gh/moby/vpnkit)
- [Windows](https://ci.appveyor.com/project/moby/vpnkit/history)

![VPNKit diagram](http://moby.github.io/vpnkit/vpnkit.png)

VPNKit is a set of tools and services for helping [HyperKit](https://github.com/moby/hyperkit)
VMs interoperate with host VPN configurations.


Building on Unix
----------------

First install `wget`, `opam` using your package manager of choice.

Build all the dependencies and the program itself with:

```
cd [path to vpnkit source]
opam remote add vpnkit ./repo/darwin
opam install --deps-only vpnkit
make
```

When the build succeeds the `vpnkit` binary should be available in the current path.

Running with hyperkit
---------------------

First ask `vpnkit` to listen for ethernet connections on a local Unix domain socket:
```
vpnkit --ethernet /tmp/ethernet --debug
```
Next ask [com.docker.hyperkit](https://github.com/moby/hyperkit) to connect a NIC to this
socket by adding a command-line option like `-s 2:0,virtio-vpnkit,path=/tmp/ethernet`. Note:
you may need to change the slot `2:0` to a free slot in your VM configuration.

Why is this needed?
-------------------

Running a VM usually involves modifying the network configuration on the host, for example
by activating Ethernet bridges, new routing table entries, DNS and firewall/NAT configurations.
Activating a VPN involves modifying the same routing tables, DNS and firewall/NAT configurations
and therefore there can be a clash -- this often results in the network connection to the VM
being disconnected.

VPNKit, part of [HyperKit](https://github.com/moby/hyperkit)
attempts to work nicely with VPN software by intercepting the VM traffic at the Ethernet level,
parsing and understanding protocols like NTP, DNS, UDP, TCP and doing the "right thing" with
respect to the host's VPN configuration.

VPNKit operates by reconstructing Ethernet traffic from the VM and translating it into the
relevant socket API calls on OSX or Windows. This allows the host application to generate
traffic without requiring low-level Ethernet bridging support.

Design
------

- [Using vpnkit as a default gateway](docs/ethernet.md): describes the flow of ethernet traffic to/from the VM
- [Port forwarding](docs/ports.md): describes how ports are forwarded from the host into the VM
- [Experimental transparent HTTP proxy](docs/transparent-http-proxy.md): describes the
  experimental support for transparent HTTP(S) proxying

Licensing
---------

VPNKit is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/moby/vpnkit/blob/master/LICENSE.md) for the full
license text.

Contributions are welcome under the terms of this license. You may wish to browse
the [weekly reports](reports) to read about overall activity in the repository.
