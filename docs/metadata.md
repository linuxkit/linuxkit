# Metadata and Userdata handling

Most providers offer two general mechanisms to provide compute instances
with information about the instance that cannot be discovered by any other
means. There are usually two types of information, namely _metadata_ and
_user-data_.  Metadata is usually set by the provider (e.g. geographical
region of the datacentre, name given to the instance, external IP address,
tags and other similar information), while userdata is fully custom,
hence the name, and it is the information that user may supply to their
instances before launch (it is immutable in most providers).

The [metadata package](../pkg/metadata/) handles both metadata and
userdata for a number of providers (see below).  It abstracts over
the provider differences by exposing both metadata and userdata in
a directory hierarchy under `/run/config`.  For example, sshd config
files from the metadata are placed under `/run/config/ssh`.

Userdata is assumed to be a single string and the contents will be
stored under `/run/config/userdata`.  If userdata is a JSON file, the
contents will be further processed, where different keys cause
directories to be created and the directories are populated with files.
For example, the following userdata file:
```JSON
{
  "ssh": {
    "entries": {
      "sshd_config": {
        "perm": "0600",
        "content": "PermitRootLogin yes\nPasswordAuthentication no"
      }
    }
  },
  "foo": {
    "entries": {
      "bar": {
        "content": "foobar"
      },
      "baz": {
        "perm": "0600",
        "content": "bar"
      }
    }
  }
}
```
will generate the following files:
```
/run/config/ssh/sshd_config
/run/config/foo/bar
/run/config/foo/baz
```

The JSON file consists of a map from `name` to an entry object. Each entry object has the following fields:
- `content`: if present then the entry is a file. The value is a string containing the desired contents of the file.
- `entries`: if present then the entry is a directory. The value is a map from `name` to entry objects.
- `perm`: the permissions to create the file with.

The `content` and `entries` fields are mutually exclusive, it is an error to include both,
one or the other _must_ be present.
The file or directory's name in each case is the same as the key which referred to that entry.

This hierarchy can then be used by individual containers, who can bind
mount the config sub-directory into their namespace where it is
needed.

## A note on SSH

Supported providers will extract public keys from metadata to a file
located at `/run/config/ssh/authorized_keys`.  You must bind this path
into the `sshd` namespace in order to make use of these keys.  Use a
configuration similar to the one shown below to enable root login
based on keys from the metadata service:

```
  - name: sshd
    image: linuxkit/sshd:4696ba61c3ec091328e1c14857d77e675802342f
    binds.add:
     - /run/config/ssh/authorized_keys:/root/.ssh/authorized_keys
```

# Metadata image creation

`linuxkit run` backends accept two options to pass metadata to the VM in a platform specific
manner to be picked up by the `pkg/metadata` component:

* `-data=STRING` will cause the given `STRING` to be passed to the VM
* `-data-file=PATH` will cause the contents of the file at `PATH` to be passed to the VM


Alternatively `linuxkit metadata create meta.iso STRING` will produce
a correctly formatted ISO image which can be passed to a VM as a CDROM
device for consumption by the `pkg/metadata` component.

# Providers

Below is a list of supported providers and notes on what is supported. We will add more over time.


## GCP

GCP metadata is reached via a well known URL
(`http://metadata.google.internal/`) and currently
we extract the hostname and populate the
`/run/config/ssh/authorized_keys` from metadata. In the future we'll
add more complete SSH support.

GCP userdata is extracted from `/computeMetadata/v1/instance/attributes/userdata`
and made available in `/run/config/userdata`.

## AWS

AWS metadata is reached via the following URL
(`http://169.254.169.254/latest/meta-data/`) and currently we extract the
hostname and populate the `/run/config/ssh/authorized_keys` from metadata.

AWS userdata is extracted from `http://169.254.169.254/latest/user-data` and
and made available in `/run/config/userdata`.

## Hetzner

Hetzner metadata is reached via the following URL
(`http://169.254.169.254/latest/meta-data/`) and currently we extract the
hostname and populate the `/run/config/ssh/authorized_keys` from metadata.

Hetzner userdata is extracted from `http://169.254.169.254/latest/user-data` and
and made available in `/run/config/userdata`.

## HyperKit

HyperKit does not distinguish metadata and userdata, it's simply
refered to as data, which is passed to the VM as a disk image
in ISO9660 format.

## Virtualization.Framework

Virtualization.Framework does not distinguish metadata and userdata, it's simply
refered to as data, which is passed to the VM as a disk image
in ISO9660 format.
