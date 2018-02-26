# Docker for Mac

[`docker-for-mac.yml`](./docker-for-mac.yml) contains an example use
of the open source components of Docker for Mac. The example has
support for controlling `dockerd` from the host via `vsudd` and port
forwarding with VPNKit. It requires HyperKit, VPNKit and a Docker
client on the host to run. The easiest way to install these at the
moment is to install a recent version of Docker for Mac.

To build it with the latest Docker CE:

```
$ linuxkit build docker-for-mac.yml
```

To run the VM with a 4G disk:

```
linuxkit run hyperkit -networking=vpnkit -vsock-ports=2376 -disk size=4096M -data-file ./metadata.json docker-for-mac
```

Where the file `./metadata.json` should contain the desired docker daemon
configuration, for example:

```
{
  "docker": {
    "entries": {
      "daemon.json": {
        "content": "{\n  \"debug\" : true,\n  \"experimental\" : true\n}\n"
      }
    }
  }
}
```

In another terminal you should now be able to access docker via the
socket `guest.00000947` in the state directory
(`docker-for-mac-state/` by default):

```
$ docker -H unix://docker-for-mac-state/guest.00000948 ps
CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
```
