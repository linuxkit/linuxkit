# External Disk
`linuxkit run` has the ability to mount an external disk when booting. It involves two steps:

1. Make the disk available as a device
2. Mount the disk

## Make Disk Available
In order to make the disk available, you need to tell `linuxkit` where the disk file or block device is.

All local `linuxkit run` methods (currently `hyperkit`, `qemu`, and `vmware`) take two arguments:

* `-disk-size <size>`: size of disk. The default is in MB but a `G` can be appended to specify the size in GB, e.g. `-disk-size 4096` or `disk-size 4G` will provide a 4GB disk.
* `-disk <path>`: use the disk at location _path_, e.g. `-disk foo.img` will use the disk at `$PWD/foo.img`

If you do not provide `-disk `_path_, `linuxkit` assumes a default, which is _prefix_`-state/disk.img` for `hyperkit` and `vmware` and _prefix_`-disk.img` for `qemu`. 

If the disk at `<path>`, or the default if `-disk` option is not provided, does not exist, `linuxkit` will create one of size `<size>`.

**TODO:** GCP

## Mount the Disk
A disk created or used via `hyperkit run` will be available inside the image at `/dev/vda` with the first partition at `/dev/vda1`.

In order to use the disk, you need to do several steps to make it available:

1. Create a partition table if it does not have one.
2. Create a filesystem if it does not have one.
3. `fsck` the filesystem.
4. Mount it.

To simplify the process, two `onboot` images are available for you to use:

1. `format`, which:
    * checks for a partition table and creates one if necessary
    * checks for a filesystem on the partition and creates one if necessary
    * runs `fsck` on the filesystem
2. `mount` which mounts the filesystem to a provided path

```yml
onboot:
  - name: format
    image: "linuxkit/format:fdbfda789fe30a97ff194a06ac51ee0ff6b3ccf4"
    binds:
     - /dev:/dev
    capabilities:
     - CAP_SYS_ADMIN
     - CAP_MKNOD
  - name: mount
    image: "linuxkit/mount:ad138d252798d9d0d6779f7f4d35b7fbcbbeefb9"
    binds:
     - /dev:/dev
     - /var:/var:rshared,rbind
    capabilities:
     - CAP_SYS_ADMIN
    rootfsPropagation: shared
    command: ["/mount.sh", "/var/external"]
```

Notice several key points:

1. format container
    * The format container needs to have bind mounts for `/dev`
    * The format container needs `CAP_SYS_ADMIN` and `CAP_MKNOD` capabilities
    * The format container only needs to run **once**, not matter how many external disks or partitions are provided. It finds all block devices under `/dev` and processes them.
2. mount container
    * The mount container `command` is `mount.sh` followed by the desired mount point. Remember that nearly everything in a linuxkit image is read-only except under `/var`, so mount it there.
    * The mount container needs to have bind mounts for `/dev` and `/var`
    * The mount container needs `CAP_SYS_ADMIN` capabilities
    * The mount container needs `rootfsPropagation: shared`

With the above in place, if run with the current disk options, the image will make the external disk available as `/dev/vda1` and mount it at `/var/external`.
