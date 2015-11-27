Workarounds
===========

## Connection Issues

There are currently a few connections issues in Boot2docker. If you see

    Couldn't connect to Docker daemon

or something like

    FATA[0000] An error occurred trying to connect: Get https://192.168.59.107:2376/v1.18/containers/json?all=1: x509: certificate is valid for 127.0.0.1, 10.0.2.15, not 192.168.59.107


it could be one of many issues reported recently.  Try this


    boot2docker ssh sudo /etc/init.d/docker restart


and run `docker ps` to see if you can connect. If not, then try deleting the keys


    boot2docker ssh sudo rm /var/lib/boot2docker/tls/server*.pem
    boot2docker ssh sudo /etc/init.d/docker restart
    boot2docker shellinit

and run `docker ps` to see if you can connect. If that doesn't work:

- `boot2docker restart`
- `docker ps` # Should see some output, not errors.




## Port forwarding

> **Note**: these instructions are for TCP only, not UDP. If you need to port forward
> UDP packets, the commands are similar. Please see the [VirtualBox
> NAT documentation](https://www.virtualbox.org/manual/ch06.html#network_nat)
> for more details.

Let's say your Docker container exposes the port 8000 and you want access it from
your other computers on your LAN. You can do it temporarily, using `ssh`:

Run following command (and keep it open) to use ssh to forward all accesses
to your OSX/Windows box's port 8000 to the Boot2Docker virtual machine's port
8000:

```sh
$ boot2docker ssh -vnNTL *:8000:localhost:8000
```

or you can set up a permanent VirtualBox NAT Port forwarding:

```sh
$ VBoxManage modifyvm "boot2docker-vm" --natpf1 "tcp-port8000,tcp,,8000,,8000";
```

If the vm is already running, you should run this other command:

```sh
$ VBoxManage controlvm "boot2docker-vm" natpf1 "tcp-port8000,tcp,,8000,,8000";
```

Now you can access your container from your host machine under `localhost:8000`.

## Port forwarding on steroids

If you use a lot of containers which expose the same port, you have to use docker dynamic port forwarding.

For example, running 3 **nginx** containers:

 - container-1 : 80 -> 49153 (i.e. `docker run -p 49153:80 ...`)
 - container-2 : 80 -> 49154 (i.e. `docker run -p 49154:80 ...`)
 - container-3 : 80 -> 49155 (i.e. `docker run -p 49155:80 ...`)

By using the `VBoxManage modifyvm` command of VirtualBox you can forward all 49XXX ports to your host. This way you can easily access all 3 webservers in you browser, without any ssh localforwarding hack. Here's how it looks like:

``` sh
# vm must be powered off
for i in {49000..49900}; do
 VBoxManage modifyvm "boot2docker-vm" --natpf1 "tcp-port$i,tcp,,$i,,$i";
 VBoxManage modifyvm "boot2docker-vm" --natpf1 "udp-port$i,udp,,$i,,$i";
done
```

This makes `container-1` accessible at `localhost:49153`, and so on.

In order to reverse this change, you can do:

``` sh
# vm must be powered off
for i in {49000..49900}; do
 VBoxManage modifyvm "boot2docker-vm" --natpf1 delete "tcp-port$i";
 VBoxManage modifyvm "boot2docker-vm" --natpf1 delete "udp-port$i";
done
```

## BTRFS (ie, mkfs inside a privileged container)

Note: AUFS on top of BTRFS has many, many issues, so the Docker engine's init script
will autodetect that `/var/lib/docker` is a `btrfs` partition and will set `-s btrfs`
for you.

```console
docker@boot2docker:~$ docker pull debian:latest
Pulling repository debian
...
docker@boot2docker:~$ docker run -i -t --rm --privileged -v /dev:/hostdev debian bash
root@5c3507fcae63:/# fdisk /hostdev/sda # if you need to partition your disk
Command: o
Command: n
Select: p
Partition: <enter>
First sector: <enter>
Last sector: <enter>
Command: w
root@5c3507fcae63:/# apt-get update && apt-get install btrfs-tools
...
The following NEW package will be installed:
  btrfs-tools
...
Setting up btrfs-tools (...) ...
root@5c3507fcae63:/# mkfs.btrfs -L boot2docker-data /hostdev/sda1
...
fs created label boot2docker-data on /hostdev/sda1
...
root@5c3507fcae63:/# exit
docker@boot2docker:~$ sudo reboot
```
