# FAQ

## I've just installed a new Boot2Docker and I get `client and server don't have the same version`

There's a good chance that your Boot2Docker virtual machine existed before you
upgraded your Docker client - what you need to run, is `boot2docker upgrade`.

For example, on Windows, having just installed using the 1.6.0 installer, I
had the following:

```
    export DOCKER_TLS_VERIFY=1

You can now use `docker` directly, or `boot2docker ssh` to log into the VM.
Welcome to Git (version 1.9.4-preview20140929)


Run 'git help git' to display the help index.
Run 'git help <command>' to display help for specific commands.

svend_000@BIG ~
$ docker info
FATA[0000] Error response from daemon: client and server don't have same version
 (client : 1.18, server: 1.17)

svend_000@BIG ~
$ boot2docker.exe upgrade
boot2docker is up to date (v1.6.0), skipping upgrade...
Latest release for github.com/boot2docker/boot2docker is v1.6.0
Downloading boot2docker ISO image...
Success: downloaded https://github.com/boot2docker/boot2docker/releases/download
/v1.6.0/boot2docker.iso
        to C:\Users\svend_000\.boot2docker\boot2docker.iso
Waiting for VM and Docker daemon to start...
...............ooo
Started.
Writing C:\Users\svend_000\.boot2docker\certs\boot2docker-vm\ca.pem
Writing C:\Users\svend_000\.boot2docker\certs\boot2docker-vm\cert.pem
Writing C:\Users\svend_000\.boot2docker\certs\boot2docker-vm\key.pem

To connect the Docker client to the Docker daemon, please set:
    export DOCKER_CERT_PATH='C:\Users\svend_000\.boot2docker\certs\boot2docker-v
m'
    export DOCKER_TLS_VERIFY=1
    export DOCKER_HOST=tcp://192.168.59.103:2376


svend_000@BIG ~
$ docker info
Containers: 0
Images: 6
Storage Driver: aufs
 Root Dir: /mnt/sda1/var/lib/docker/aufs
 Backing Filesystem: extfs
 Dirs: 6
 Dirperm1 Supported: true
Execution Driver: native-0.2
Kernel Version: 3.18.11-tinycore64
Operating System: Boot2Docker 1.6.0 (TCL 5.4); master : a270c71 - Thu Apr 16 19:
50:36 UTC 2015
CPUs: 8
Total Memory: 1.961 GiB
Name: boot2docker
ID: JGXX:ZFVX:WNJX:SSNB:QHU6:FP7P:VFDJ:EE3J:ZRYU:X3IR:6BD2:BEWM
Debug mode (server): true
Debug mode (client): false
Fds: 11
Goroutines: 16
System Time: Tue Apr 28 01:52:11 UTC 2015
EventsListeners: 0
Init SHA1: 9145575052383dbf64cede3bac278606472e027c
Init Path: /usr/local/bin/docker
Docker Root Dir: /mnt/sda1/var/lib/docker
```

## What are the specs of the VM?

* CPU Cores: same as host (physical, not logical)
* 40gb HDD (auto-initialized at first boot)
* 2GB memory
* Autoboots to Boot2Docker
* `virtio` high performance networking
* NAT networked (Docker `2375->2375` and SSH `22->2022` are forwarded to the host)

## How can I solve my problems with SSH?

If `ssh` complains about the keys:

```
$ ssh-keygen -R '[localhost]:2022'
```

## Login as root

Run `sudo -s` as the docker user.

## What is the Boot2Docker distribution based on?

It is based on a stripped down [Tiny Core Linux](http://tinycorelinux.net).

## Why not CoreOS?

I got asked that question a lot, so I thought I should put it here once and for
all. [CoreOS](http://coreos.com/) is targeted at building infrastructure and
distributed systems. I just wanted the fastest way to boot to Docker.

## Persistent partition choice

Boot2Docker will first try to mount a partition labeled ``boot2docker-data``, if
that doesn't exist, it will pick the first ``ext4`` partition listed by ``blkid``.

## Local Customisation (with persistent partition)

Changes outside of the `/var/lib/docker` and `/var/lib/boot2docker` directories
will be lost after powering down or restarting the boot2docker VM. However, if
you have a persistence partition (created automatically by `boot2docker init`),
you can make customisations that are run at the end of boot initialisation by
creating a script at ``/var/lib/boot2docker/bootlocal.sh``.

From Boot2Docker version 1.6.0, you can also specify steps that are run before
the Docker daemon is started, using `/var/lib/boot2docker/bootsync.sh`.

You can also set variables that will be used during the boot initialisation (after
the automount) by setting them in `/var/lib/boot2docker/profile`

For example, to download ``pipework``, install its pre-requisites (which you can
download using ``tce-load -w package.tcz``), and then start a container:

```
#!/bin/sh


if [ ! -e /var/lib/boot2docker/pipework ]; then
        curl -o /var/lib/boot2docker/pipework https://raw.github.com/jpetazzo/pipework/master/pipework
        chmod 777 /var/lib/boot2docker/pipework
fi

#need ftp://ftp.nl.netbsd.org/vol/2/metalab/distributions/tinycorelinux/4.x/x86/tcz/bridge-utils.tcz
#and iproute2 (and its friends)
su - docker -c "tce-load -i /var/lib/boot2docker/*.tcz"

#start my management container if its not already there
docker run -d -v /var/run/docker.sock:/var/run/docker.sock $(which docker):$(which docker)  -name dom0 svens-dom0
```

Or, if you need to tell the Docker daemon to use a specific DNS server, add the 
following to ``/var/lib/boot2docker/profile``:

```
EXTRA_ARGS="--dns 192.168.1.2"
```

## What is the development process

We are implementing the same process as [Docker merge approval](
https://github.com/dotcloud/docker/blob/master/CONTRIBUTING.md#merge-approval),
so all commits need to be done via pull requests, and will need 2 or more LGTMs.

## Is boot2docker only for VirtualBox?

There are two parts of Boot2Docker: the ISO image, and the `boot2docker` management
tool to set up and manage a VM. The management tool only works with VirtualBox,
but the ISO image is designed to also be used with physical hardware. There
are no plans to make separate ISO images for different configurations.
