# SSH example

The LinuxKit [sshd example](./sshd.yaml) defines an image running a SSH
daemon. You can build it as usual (though you should add your public
key to the `contents` field in the `files` section).

On some platforms you can then just ssh into the system once it is running, but on some platforms additional steps are required.


## HyperKit/Docker for Mac

If you use the HyperKit backend with Docker for Mac, the VM created with `moby run ...` is placed on the same network as the Docker for Mac VM (via VPNKit). 
The VMs network is not directly accessible from the host, but is accessible from within containers run with Docker for Mac.

So, to ssh into the VM created via `moby run sshd` it's best to do this via a container from within a container.

You can build a small container with an ssh client with this Dockerfile:
```
FROM alpine:edge
RUN apk add --no-cache openssh-client
```
Then:
```
docker build -t ssh .
```

And now:
```
docker run --rm -ti -v ~/.ssh:/root/.ssh  ssh ssh <IP address of VM>
```

The HyperKit backend for `moby run` also allows you to set the IP address of the VM, like:
```
moby run -ip 192.168.65.101 sshd
```


## Qemu/Linux

TBD

