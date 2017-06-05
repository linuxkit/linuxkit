### Tools for Docker for Desktop

This directory contains open source tools that are used in Docker for Desktop while they are being ported to LinuxKit. 

 - `vsudd` is a vsock to unix domain socket forwarding daemon
 - `proxy` contains network forwarding proxies (proxy-vsockd for incoming, slirp-proxy for outgoing)
 - `docker-dfm` is the docker daemon bundled with the outgoing vpnkit proxy
 - `configure-dfm` is an onboot container that mounts /dfm/port and sets up a new /var/lib/docker 
 - `install-dfm` is an init container that initially sets up /dfm

More components will be added in the future as we port Docker for Desktop to LinuxKit.

#### Docker example
Docker should mostly work on mac, but volume sharing is not supported yet.

An example configuration file is available in [examples/dfm.yml](examples/dfm.yml).

When running the example configuration, VSOCK ports 2373 (logging), 2374 (containerd) and 2375 (dockerd) should be
forwarded to the host. To enable container port forwarding you have to use the `vpnkit` networking mode. You also need a 
disk image for `/var/lib/docker`.

Example:
```
$ moby build dfm 
...
$ linuxkit run hyperkit -disk-size 250 -vsock-ports 2373,2374,2375 -networking=vpnkit dfm
```

You should now be able to connect with a Docker client on the host. Unix domain sockets corresponding to the forwarded VSOCK 
ports should be in the state directory. Port 2375 (docker) is `guest.00000947`.

Forward a port with nginx:
```
$ docker -H unix://guest.00000947 run -p 8080:80 -it --rm nginx
```

Port should now be open on the host:
```
$ curl localhost:8080
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...
```

You can also tail the logs of the LinuxKit system containers and dockerd containers from the host:

```
$ go get -u -v -ldflags -s github.com/linuxkit/linuxkit/projects/logging/pkg/memlogd/cmd/logread
...
$ logread -F -socket guest.00000945
2017-05-22T18:57:36Z memlogd memlogd started
2017-05-22T18:57:36Z 002-dhcpcd.stdout eth0: waiting for carrier
2017-05-22T18:57:36Z 002-dhcpcd.stderr eth0: could not detect a useable init system
2017-05-22T18:57:36Z 002-dhcpcd.stdout eth0: carrier acquired
2017-05-22T18:57:36Z 002-dhcpcd.stdout DUID 00:01:00:01:20:b5:f1:20:02:50:00:00:00:0d
2017-05-22T18:57:36Z 002-dhcpcd.stdout eth0: IAID 00:00:00:0d
2017-05-22T18:57:36Z 002-dhcpcd.stdout eth0: adding address fe80::d539:c1e:1b30:bee0
...
2017-05-31T09:40:29Z e0ef41ca92179b0bc6e8668ef8bb98653a69242315b4e40a783e9faf868d2356.stdout 172.18.0.1 - - [31/May/2017:09:40:29 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.51.0" "-"
```

You should also be able to use `ctr` to manage `containerd` as described in the vsudd example below.

#### Vsudd example

Vsudd is used to forward unix domain socket traffic from the host to the guest VM with VSOCK. It's currently only working on 
Mac. An example configuration file is available in [examples/vsudd.yml](examples/vsudd.yml).

After building the example, run the example with `linuxkit run hyperkit -vsock-ports 2373,2374 vsudd`. This will create 
two unix domain sockets in the state directory that map to the `containerd` and `memlogd` control sockets. The sockets are 
called `guest.00000945` and `guest.00000946`.

If you install the `ctr` tool on the host you should be able to access the `containerd` running in the VM:

```
$ go get -u -v -ldflags -s github.com/containerd/containerd/cmd/ctr
...
$ ctr -a guest.00000946 list
ID        PID       STATUS
rngd      642       RUNNING
vsudd     693       RUNNING
```

The `onboot` and `service` container logs can be read from the host if you install the `logread` tool, as in the Docker example.

