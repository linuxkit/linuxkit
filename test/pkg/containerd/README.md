# containerd test

This is a test package for containerd. It is expected to run inside a containerd on a system
that can make filesystems such as `mkfs.ext4`, create devmapper devices such as `dmsetup`, and
do other root level things.

To run this in a container, you need:

* `/dev:/dev` mounted
* `/tmp:/tmp` mounted
* `/var/lib/containerd-test` as tmpfs
* container is privileged

The [build.yml](build.yml) file contains the necessary mounts and capabilities.

To run this standalone and perform the test, you can:

1. `docker build -t containerd-test .`
2. `docker run --rm --privileged -v /dev:/dev -v /tmp:/tmp --tmpfs /var/lib/containerd-test containerd-test`
