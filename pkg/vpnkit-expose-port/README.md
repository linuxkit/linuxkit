### vpnkit-expose-port

This init-package provides `vpnkit-expose-port` and `vpnkit-iptables-wrapper` from [vpnkit](http://github.com/moby/vpnkit.git). The binaries are installed on the host in `/usr/local/bin` and can be bind mounted into a container with `dockerd`.

`vpnkit-expose-port` is a userland proxy that opens ports on the host by demand. To enable it, start `dockerd` with `--userland-proxy-path` pointing to the bind mounted binary.

`vpnkit-iptables-wrapper` is a wrapper for iptables that opens ports via vpnkit for swarm services. It has to be bind mounted as `iptables` in $PATH before the regular `iptables` binary.

To coordinate with `vpnkit` both tools require access to the 9P port configuration mount point.
