# External Disk
`linuxkit run` has the ability to mount an external disk when booting. It involves two steps:

1. Make the disk available as a device
2. Mount the disk

## Make Disk Available
In order to make the disk available, you need to tell `linuxkit` where the disk file or block device is.

All local `linuxkit run` methods (currently `hyperkit`, `qemu`, and `vmware`) take a `-disk` argument:

* `-disk path,size=100M,format=qcow2`. For size the default is in GB but an `M` can be appended to specify sizes in MB. The format can be omitted for the platform default, and is only useful on `qemu` at present.

If a _path_ is specified `linuxkit` will use the disk at location _path_. If you do not provide `-disk ` _path_, `linuxkit` assumes a default path, which is _prefix_`-state/disk.img`. 

If the disk at the specified or default _path_ does not exist, `linuxkit` will create one of size _size_.

The `-disk` specification may be repeated for multiple disks, although a limited number may be supported, and some platforms currently only support a single disk.

**TODO:** GCP

## Format the disk

`pkg/format` creates a partition table and format drives for use with LinuxKit

### Example Usage

This packages supports two modes of use:

```
onboot:
  - name: format
    image: linuxkit/format:<hash>
```

In this mode of operation, the first disk found that does not have a valid partition table
will have one linux partition created that fills the entire disk

### Options

```
onboot:
  - name: format
    image: linuxkit/format:<hash>
    command: ["/usr/bin/format", "-type", "ext4", "-label", "DATA", "/dev/vda"]
```

```
onboot:
  - name: format
    image: linuxkit/format:<hash>
    command: ["/usr/bin/format", "-force", "-type", "xfs", "-label", "DATA", "-verbose", "/dev/vda"]
```

- `-force` can be used to force the partition to be cleared and recreated (if applicable), and the recreated partition formatted. This option would be used to re-init the partition on every boot, rather than persisting the partition between boots.
- `-label` can be used to give the disk a label
- `-type` can be used to specify the type. This is `ext4` by default but `btrfs` and `xfs` are also supported
- `-verbose` enables verbose logging, which can be used to troubleshoot device auto-detection and (re-)partitioning
- The final (optional) argument specifies the device name

## Mount the disk

Once a disk has been prepared it will need to be mounted using `pkg/mount`

### Usage

**NOTE: Block devices may only be mounted in `/var` unless you have explicitly added an additional bind mount**

If no additional arguments are provided the first unmounted linux partition on the first block device is mounted to the mountpoint provided.

```
onboot:
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "/var/lib/docker"]
```

### Options

You can provide either a partition label, device name or disk UUID to specify which disk should be used.
For example:

```
onboot:
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "-label", "DATA", "/var/lib/docker" ]
```

```
onboot:
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "-uuid", "a-proper-uuid", "/var/lib/docker" ]
```

```
onboot:
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "-device", "/dev/sda1", "/var/lib/docker" ]
```

For compatibility with the standard `mount` command we also support providing the device name as a positional argument.
E.g

```
onboot:
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "/dev/sda1", "/var/lib/docker" ]
```

## Extending Partitions

`pkg/extend` can extends a single partition to fill the entire disk

### Usage

In the default mode of operation, any disks that are found and have a single partition and free space will have that partition extended.

```
onboot:
  - name: extend
    image: linuxkit/extend:<hash>
```

### Options

`-type` can be used to specify the type. The default is `ext4` but `btrfs` and `xfs` are also supported.
If you know the name of the disk that you wish to extend you may supply this as an argument

```
onboot:
  - name: extend
    image: linuxkit/extend:<hash>
    command: ["/usr/bin/extend", "-type", "btrfs", "/dev/vda"]
```

