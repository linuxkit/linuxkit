# Boot2Docker

Boot2Docker is a lightweight Linux distribution made specifically to run
[Docker](https://www.docker.com/) containers. It runs completely from RAM, is a
small ~24MB download and boots in ~5s (YMMV).

## Features

* Kernel 4.1.10 with AUFS, Docker v1.8.2 - using libcontainer
* Container persistence via disk automount on `/var/lib/docker`
* SSH keys persistence via disk automount

> **Note:** Boot2Docker uses port **2376**, the [registered IANA Docker SSL
> port](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml?search=docker)

## Caveat Emptor

Boot2Docker is currently designed and tuned for development.  Using it for
any kind of production workloads at this time is highly discouraged.

## Installation

Installation instructions for [OS X](https://docs.docker.com/installation/mac/)
and [Windows](https://docs.docker.com/installation/windows/) are available on
the Docker documentation site.

The [ISO can be downloaded
here](https://github.com/boot2docker/boot2docker/releases).

### All in one Installers for OS X and Windows

We have built installers for [OS
X](https://github.com/boot2docker/osx-installer/releases) and
[Windows](https://github.com/boot2docker/windows-installer/releases) which will
install the `boot2docker` management tool, VirtualBox, and any tools needed to
run Boot2Docker.

### Installation using the `boot2docker` management tool

If you have the prerequisites, or want to help develop Boot2Docker, you can also
download the appropriate [boot2docker management
release](https://github.com/boot2docker/boot2docker-cli/releases) and use it to
download
[`boot2docker.iso`](https://github.com/boot2docker/boot2docker/releases).

## How to use

The `boot2docker` management tool leverages VirtualBox's `VBoxManage` to
initialise, start, stop and delete the VM right from the command line.

#### Initialize

```console
$ boot2docker init
```

#### Start VM

```console
$ boot2docker up
```

#### Upgrade the Boot2docker VM image

```console
$ boot2docker stop
$ boot2docker download
$ boot2docker up
```

If your Boot2Docker virtual machine was created prior to 0.11.1-pre1, it's best
to delete -  `boot2docker delete` and then `boot2docker init` to create a new
VM.

The main changes are to add a `/var/lib/boot2docker/userdata.tar` file that is
un-tarred into the `/home/docker` directory on boot. This file contains a
`.ssh/authorized_keys` and `.ssh/authorized_keys2` files containing a public
SSH key.

## Docker Hub

To save and share container images, automate workflows, and more sign-up for a
free [Docker Hub account](https://hub.docker.com).

## More information

See [Frequently asked questions](doc/FAQ.md) for more details.

#### Boot script log

The bootup script output is logged to `/boot.log`, so you can see (and
potentially debug) what happens. Note that this is not persistent between boots
because we're logging from before the persistence partition is mounted (and it
may not exist at all).

#### Container Port redirection

The latest version of `boot2docker` sets up two network adaptors, one using NAT
to allow the VM to download images and files from the internet, and a host only
network that Docker container's ports will be exposed on.

If you run a container with an exposed port, and then use OSX's `open` command:

```console
$ boot2docker up
$ eval "$(boot2docker shellinit)"
$ docker run --name nginx-test -d -p 80:80 nginx
$ open http://$(boot2docker ip 2>/dev/null)/
$ docker stop nginx-test
$ docker rm nginx-test
```

The `eval "$(boot2docker shellinit)"` sets the `DOCKER_HOST` environment variable for
this shell, then the `docker run` starts the webserver as a daemon, and `open`
will then show the default page in your default web browser (using `boot2docker
ip`).

If you want to share container ports with other computers on your LAN, you will
need to set up [NAT adaptor based port forwarding](doc/WORKAROUNDS.md).

#### Docker daemon options

If you need to customize the options used to start the Docker daemon, you can
do so by adding entries to the `/var/lib/boot2docker/profile` file on the
persistent partition inside the Boot2Docker virtual machine. Then restart the
daemon.

The following example will enable core dumps inside containers, but you can
specify any other options you may need.

```console
boot2docker ssh -t sudo vi /var/lib/boot2docker/profile
# Add something like:
#     EXTRA_ARGS="--default-ulimit core=-1"
boot2docker restart
```

#### TLS support

By default, `boot2docker` runs `docker` with TLS enabled. It auto-generates
certificates and stores them in `/home/docker/.docker` inside the VM. The
`boot2docker up` command will copy them to `~/.boot2docker/certs` on the
host machine once the VM has started, and output the correct values for
the `DOCKER_CERT_PATH` and `DOCKER_TLS_VERIFY` environment variables.

`eval "$(boot2docker shellinit)"` will also set them correctly.

We strongly recommend against running Boot2Docker with an unencrypted Docker
socket for security reasons, but if you have tools that cannot be easily
switched, you can disable it by adding `DOCKER_TLS=no` to your
`/var/lib/boot2docker/profile` file.

#### Folder sharing

Boot2Docker is essentially a remote Docker engine with a read only filesystem
(other than Docker images, containers and volumes). The most scalable and
portable way to share disk space between your local desktop and a Docker
container is by creating a volume container and then sharing that to where it's
needed.

One well tested approach is to use a file sharing container like
`svendowideit/samba`:

```console
$ # Make a volume container (only need to do this once)
$ docker run -v /data --name my-data busybox true
$ # Share it using Samba (Windows file sharing)
$ docker run --rm -v /usr/local/bin/docker:/docker -v /var/run/docker.sock:/docker.sock svendowideit/samba my-data
$ # then find out the IP address of your Boot2Docker host
$ boot2docker ip
192.168.59.103
```

Connect to the shared folder using Finder (OS X):

	Connect to cifs://192.168.59.103/data
	Once mounted, will appear as /Volumes/data

Or on Windows, use Explorer to Connect to:

	\\192.168.59.103\data

You can then use your data container from any container you like:

```console
$ docker run -it --volumes-from my-data ubuntu
```

You will find the "data" volume mounted as "/data" in that container. Note that
"my-data" is the name of volume container, this is shared via the "network" by
the "samba" container that refers to it by name. So, in this example, if you
were on OS-X you now have /Volumes/data and /data in container being shared. You
can change the paths as needed.

##### VirtualBox Guest Additions

Alternatively, Boot2Docker includes the VirtualBox Guest Additions built in for
the express purpose of using VirtualBox folder sharing.

The first of the following share names that exists (if any) will be
automatically mounted at the location specified:

1. `Users` share at `/Users`
2. `/Users` share at `/Users`
3. `c/Users` share at `/c/Users`
4. `/c/Users` share at `/c/Users`
5. `c:/Users` share at `/c/Users`

If some other path or share is desired, it can be mounted at run time by doing
something like:

```console
$ mount -t vboxsf -o uid=1000,gid=50 your-other-share-name /some/mount/location
```

It is also important to note that in the future, the plan is to have any share
which is created in VirtualBox with the "automount" flag turned on be mounted
during boot at the directory of the share name (ie, a share named `home/jsmith`
would be automounted at `/home/jsmith`).

In case it isn't already clear, the Linux host support here is currently hazy.
You can share your `/home` or `/home/jsmith` directory as `Users` or one of the
other supported automount locations listed above, but note that you will then
need to manually convert your `docker run -v /home/...:...` bind-mount host
paths accordingly (ie, `docker run -v /Users/...:...`).  As noted in the
previous paragraph however, this is likely to change in the future as soon as a
more suitable/scalable solution is found and implemented.


#### Installing secure Registry certificates

You can add your Registry server's public certificate (in `.pem` format) into
the `/var/lib/boot2docker/certs/` directory, and Boot2Docker will automatically
load it from the persistence partition at boot.

You may need to add several certificates (as separate `.pem` files) to this
directory, depending on the CA signing chain used for your certificate.

##### Insecure Registry

As of Docker version 1.3.1, if your registry doesn't support HTTPS, you must add it as an
insecure registry.

```console
$ boot2docker init
$ boot2docker up
$ boot2docker ssh "echo $'EXTRA_ARGS=\"--insecure-registry <YOUR INSECURE HOST>\"' | sudo tee -a /var/lib/boot2docker/profile && sudo /etc/init.d/docker restart"
```

then you should be able to do a docker push/pull.

#### Running behind a VPN (Cisco AnyConnect, etc)

So sometimes if you are behind a VPN, you'll get an `i/o timeout` error.
The current work around is to forward the port in the boot2docker-vm.

If you get an error like the following:

```no-highlight
Sending build context to Docker daemon
2014/11/19 13:53:33 Post https://192.168.59.103:2376/v1.15/build?rm=1&t=your-tag: dial tcp 192.168.59.103:2376: i/o timeout
```

That means you have to forward port `2376`, which can be done like so:

* Open VirtualBox
* Open Settings > Network for your 'boot2docker-vm'
* Select the adapter that is 'Attached To': 'NAT' and click 'Port Forwarding'.
* Add a new rule:
	- Protocol: TCP
	- Host IP: 127.0.0.1
	- Host Port: 5555
	- Guest Port: 2376
* Set `DOCKER_HOST` to 'tcp://127.0.0.1:5555'

#### Customize

The `boot2docker` management tool allows you to customise many options from both
the command line, or by setting them in its configuration file.

See `boot2docker config` for more (including the format of the configuration
file).

#### SSH into VM

```console
$ boot2docker ssh
```

`boot2docker` auto logs in using the generated SSH key, but if you want to SSH
into the machine manually (or you're not using a `boot2docker` managed VM), the
credentials are:

```
user: docker
pass: tcuser
```

#### Persist data

Boot2docker uses [Tiny Core Linux](http://tinycorelinux.net), which runs from
RAM and so does not persist filesystem changes by default.

When you run `boot2docker init`, the `boot2docker` tool auto-creates a disk that
will be automounted and used to persist your docker data in `/var/lib/docker`
and `/var/lib/boot2docker`.  This virtual disk will be removed when you run
`boot2docker delete`.  It will also persist the SSH keys of the machine.
Changes outside of these directories will be lost after powering down or
restarting the VM - to make permanent modifications see the
[FAQ](doc/FAQ.md#local-customisation-with-persistent-partition).

If you are not using the `boot2docker` management tool, you can create an `ext4`
or `btrfs` formatted partition with the label `boot2docker-data` (`mkfs.ext4 -L
boot2docker-data /dev/sdX5`) to your VM or host, and Boot2Docker will automount
it on `/mnt/sdX` and then softlink `/mnt/sdX/var/lib/docker` to
`/var/lib/docker`.

#### Install on any device

To 'install' the ISO onto an SD card, USB-Stick or even empty hard disk, you can
use `dd if=boot2docker.iso of=/dev/sdX`.  This will create the small boot
partition, and install an MBR.

#### Build your own Boot2Docker ISO

Goto [How to build](doc/BUILD.md) for Documentation on how to build your own
Boot2Docker ISOs.

## Troubleshooting

See the [workarounds doc](https://github.com/boot2docker/boot2docker/blob/master/doc/WORKAROUNDS.md) for solutions to known issues.

#### `boot2docker up` doesn't work (OSX)

Sometimes OSX will install updates that break VirtualBox and require a restart
of the kernel extensions that boot2docker needs in order to run.  If you go to
boot boot2docker after some updates or a system restart and you get an output
such as the following:

```console
$ boot2docker up
error in run: Failed to start machine "boot2docker-vm" (run again with -v for details)
```

You may need to reload the kernel extensions in order to get your system
functioning again.

In this case, try running the following script (supplied with Virtual Box):

```console
$ sudo /Library/Application\ Support/VirtualBox/LaunchDaemons/VirtualBoxStartup.sh restart
```

You should see output such as:

```
/Applications/VirtualBox.app/Contents/MacOS/VBoxAutostart => /Applications/VirtualBox.app/Contents/MacOS/VBoxAutostart-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxBalloonCtrl => /Applications/VirtualBox.app/Contents/MacOS/VBoxBalloonCtrl-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxDD2GC.gc => /Applications/VirtualBox.app/Contents/MacOS/VBoxDD2GC.gc-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxDDGC.gc => /Applications/VirtualBox.app/Contents/MacOS/VBoxDDGC.gc-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxExtPackHelperApp => /Applications/VirtualBox.app/Contents/MacOS/VBoxExtPackHelperApp-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxHeadless => /Applications/VirtualBox.app/Contents/MacOS/VBoxHeadless-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxManage => /Applications/VirtualBox.app/Contents/MacOS/VBoxManage-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxNetAdpCtl => /Applications/VirtualBox.app/Contents/MacOS/VBoxNetAdpCtl-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxNetDHCP => /Applications/VirtualBox.app/Contents/MacOS/VBoxNetDHCP-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxNetNAT => /Applications/VirtualBox.app/Contents/MacOS/VBoxNetNAT-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxSVC => /Applications/VirtualBox.app/Contents/MacOS/VBoxSVC-amd64
/Applications/VirtualBox.app/Contents/MacOS/VBoxXPCOMIPCD => /Applications/VirtualBox.app/Contents/MacOS/VBoxXPCOMIPCD-amd64
/Applications/VirtualBox.app/Contents/MacOS/VMMGC.gc => /Applications/VirtualBox.app/Contents/MacOS/VMMGC.gc-amd64
/Applications/VirtualBox.app/Contents/MacOS/VirtualBox => /Applications/VirtualBox.app/Contents/MacOS/VirtualBox-amd64
/Applications/VirtualBox.app/Contents/MacOS/VirtualBoxVM => /Applications/VirtualBox.app/Contents/MacOS/VirtualBoxVM-amd64
/Applications/VirtualBox.app/Contents/MacOS/vboxwebsrv => /Applications/VirtualBox.app/Contents/MacOS/vboxwebsrv-amd64
Loading VBoxDrv.kext
Loading VBoxUSB.kext
Loading VBoxNetFlt.kext
Loading VBoxNetAdp.kext
```

Now the VM should boot properly.
