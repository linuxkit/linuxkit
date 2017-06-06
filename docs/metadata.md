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
a directory hierarchy under `/var/config`.  For example, sshd config
files from the metadata are placed under `/var/config/ssh`.

Userdata is assumed to be a single string and the contents will be
stored under `/var/config/userdata`.  If userdata is a JSON file, the
contents will be further processed, where different keys cause
directories to be created and the directories are populated with files.
For example, the following userdata file:
```JSON
{
    "ssh" : {
        "sshd_config" : {
            "perm" : "0600",
            "content": "PermitRootLogin yes\nPasswordAuthentication no"
        }
    },
    "foo" : {
        "bar" : "foobar",
        "baz" : {
            "perm": "0600",
            "content": "bar"
        }
    }
}
```
will generate the following files:
```
/var/config/ssh/sshd_config
/var/config/foo/bar
/var/config/foo/baz
```

Each file can either be:

- a simple string (as for `foo/bar` above) in which case the file will
  be created with the given contents and read/write (but not execute)
  permissions for user and read permissions for group and everyone else (in octal format `0644`).
- a map (as for `ssh/sshd_config` and `foo/baz` above) with the
  following mandatory keys:
  - `content`: the contents of the file.
  - `perm`: the permissions to create the file with.

This hierarchy can then be used by individual containers, who can bind
mount the config sub-directory into their namespace where it is
needed.

# Metadata image creation

Run `linuxkit run` backends accept a `--data=STRING` option which will
cause the given string to be passed to the VM in a platform specific
manner to be picked up by the `pkg/metadata` component.

Alternatively `linuxkit metadata create meta.iso STRING` will produce
a correctly formatted ISO image which can be passed to a VM as a CDROM
device for consumption by the `pkg/metadata` component.

# Providers

Below is a list of supported providers and notes on what is supported. We will add more over time.


## GCP

GCP metadata is reached via a well known URL
(`http://metadata.google.internal/`) and currently
we extract the hostname and populate the
`/var/config/ssh/authorized_keys` from metadata. In the future we'll
add more complete SSH support.

GCP userdata is extracted from `/computeMetadata/v1/instance/attributes/userdata`
and made available in `/var/config/userdata`.

## AWS

AWS metadata is reached via the following URL
(`http://169.254.169.254/latest/meta-data/`) and currently we extract the
hostname and populate the `/var/config/ssh/authorized_keys` from metadata.

AWS userdata is extracted from `http://169.254.169.254/latest/user-data` and
and made available in `/var/config/userdata`.


## HyperKit

HyperKit does not distiguish metadata and userdata, it's simply
refered to as data, which is passed to the VM as a disk image
in ISO9660 format.
