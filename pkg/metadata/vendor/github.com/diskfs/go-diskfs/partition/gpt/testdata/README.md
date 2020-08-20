# GPT Test Fixtures
This directory contains test fixtures for GUID Partition Table. Specifically, it contains the following files:

* `gpt.img`: A 10MB GPT partitioned disk with one partition.
* `gpt_partition.img`: A 128-byte subset of the disk with the bytes entry of just the one partition entry

To generate these files:

```
$ docker run -it --rm -v $PWD:/data alpine:3.6
# apk --update add sgdisk
# dd if=/dev/zero of=/data/gpt.img bs=1M count=10
# sgdisk --clear --new 1:2048:3048 --typecode=1:ef00 --change-name=1:'EFI System' --partition-guid=1:5ca3360b-5de6-4fcf-b4ce-419cee433b51 /data/gpt.img
# dd if=/dev/random of=/data/gpt.img bs=512 seek=2048 count=1000 conv=notrunc
# dd if=/data/gpt.img of=/data/gpt_partition.dat bs=1 count=128 skip=1024
# exit
$
```

You now have the exact gpt files in `$PWD`
