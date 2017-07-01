### vpnkit-forwarder

This package provides `vpnkit-forwarder` from [vpnkit](http://github.com/moby/vpnkit.git).

`vpnkit-forwarder` is a forwarding daemon used by Docker for Desktop to forward ports from Docker containers to the host via VSOCK.

To coordinate with `vpnkit` it requires access to the 9P port configuration mount point.
