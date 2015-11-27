How to build boot2docker.iso locally
================================

The boot2docker.iso is built with Docker, via a Dockerfile.

During `docker build` we
* fetch, patch with AUFS support and build the 3.15.3 Linux Kernel with Tiny Core base configuration
* build the base rootfs for boot2docker (not complete)
* build the rootfs, download the latest Docker release and create the `.iso` file on `/` of the container.

Running the resultant image will cat the iso file to STDOUT.

So the full build process goes like this:

```console
# you will need more than 2GB memory
$ docker build -t boot2docker . && docker run --rm boot2docker > boot2docker.iso
```

Now you can install the iso to a USB drive, SD card, CD-Rom or hard-disk. The image contains
a Master Boot Record, and a partition table, so can be written to a raw device.

```console
$ sudo dd if=boot2docker.iso of=/dev/sdX
```

Making your own customised boot2docker ISO
==========================================

The `boot2docker.iso` release process takes advantage of Docker Hub's
[Automated Builds](https://index.docker.io/u/boot2docker/) so
rather than modifying the `Dockerfile` and re-building from scratch,
you can make a new ``Dockerfile`` that builds ``FROM boot2docker/boot2docker``
and then run that to generate your `boot2docker.iso` file:


```console
$ sudo docker pull boot2docker/boot2docker
$ echo "FROM boot2docker/boot2docker" > Dockerfile
$ echo "ADD . $ROOTFS/data/" >> Dockerfile
$ echo "RUN somescript.sh" >> Dockerfile
$ echo "RUN /make_iso.sh" >> Dockerfile
$ echo 'CMD ["cat", "boot2docker.iso"]' >> Dockerfile

$ sudo docker build -t my-boot2docker-img .
$ sudo docker run --rm my-boot2docker-img > boot2docker.iso

```
