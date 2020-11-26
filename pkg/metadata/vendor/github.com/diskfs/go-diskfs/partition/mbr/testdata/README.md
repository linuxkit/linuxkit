# MBR Test Fixtures
This directory contains test fixtures for Master boot Record. Specifically, it contains the following files:

* `mbr.img`: A 10MB MBR partitioned disk with one partition, on which a FAT32 filesystem is embedded.
* `mbr_partition.img`: A 16-byte subset of the disk with the bytes entry of just the one partition entry

To generate these files:

```
$ docker run -it --rm -v $PWD:/data alpine:3.6
# apk --update add sfdisk dosfstools
# dd if=/dev/zero of=/data/mbr.img bs=1M count=20
# echo '2048,20480,;' | sfdisk /data/mbr.img
# dd if=/dev/zero of=/tmp/fat32.img bs=1M count=10
# mkfs.vfat -v -F 32 /tmp/fat32.img
# dd if=/tmp/fat32.img of=/data/mbr.img bs=512 seek=2048 count=20480 conv=notrunc
# dd if=/data/mbr.img of=/data/mbr_partition.dat bs=1 count=16 skip=446
# exit
$
```

You now have the exact mbr files in `$PWD`
