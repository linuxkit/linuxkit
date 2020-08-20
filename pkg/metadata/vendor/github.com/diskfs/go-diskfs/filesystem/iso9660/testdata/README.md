# ISO9660 Test Fixtures
This directory contains test fixtures for FAT32 filesystems. Specifically, it contains the following files:

* `file.iso`: A 10MB iso9660 image
* `volrecords.iso`: The volume descriptor set from a real complex distribution, specifically `Ubuntu-Server 18.04.1 LTS amd64`

To generate the `file.iso` :


```
./buildtestiso.sh
```

We make the `\foo` directory with sufficient entries to exceed a single sector (>2048 bytes). This allows us to test reading directories past a sector boundary). Since each directory entry is at least ~34 bytes + filesize name, we create 10 byte filenames, for a directory entry of 44 bytes. With a sector size of 2048 bytes, we need 2048/44 = 46 entries to fill the cluster and one more to get to the next one, so we make 50 entries.

To generate the `volrecords.iso`:

1. Download Ubuntu Server 18.0.4.1 LTS amd64 from http://releases.ubuntu.com/18.04.1/ubuntu-18.04.1-live-server-amd64.iso?_ga=2.268908601.917862151.1539151848-2128720580.1476045272
2. Copy out the desired bytes: `dd if=ubuntu-18.04.1-live-server-amd64.iso of=volrecords.iso bs=2048 count=4 skip=16`

## Utility
This directory contains a utility to output data from an ISO. It can:

* read a directory and its entries
* read a path table

To build it:

```
go build isoutil.go
```

To run it, run `./isoutil <command> <args>`. The rest of this section describes it.

### Reading an ISO directory
To read an ISO directory:

```
./isoutil directory <filename> <path>
```

Where:
* `<filename>` name of the ISO file, e.g. `file.iso`
* `<path>` absolute path to the directory, e.g. `/FOO`

### Reading an ISO path table
To read the path table:

```
./isoutil readpath <filename>
```

Where:
* `<filename>` name of the ISO file, e.g. `file.iso`
