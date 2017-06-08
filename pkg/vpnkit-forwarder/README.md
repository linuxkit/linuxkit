### vpnkit-forwarder

This package provides `vpnkit-forwarder` and `vpnkit-expose-port` from [vpnkit](http://github.com/moby/vpnkit.git).

`vpnkit-forwarder` is a forwarding daemon used by Docker for Desktop to forward ports from Docker containers to the host via VSOCK.  

`vpnkit-expose-port` is a userland proxy that opens ports by demand.

To coordinate with `vpnkit` both tools require access to the 9P port configuration mount point.
