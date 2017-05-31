docker-proxy which can set up tunnels into the VM
=================================================

This is a replacement for the built-in `docker-proxy` command, which
proxies data from external ports to internal container ports.

This program uses the 9P filesystem under /port to extend the port
forward from the host running the Moby VM all the way to the container.

docker-proxy -proto tcp -host-ip 0.0.0.0 -host-port 8080 -container-ip 172.17.0.2 -container-port 8080

