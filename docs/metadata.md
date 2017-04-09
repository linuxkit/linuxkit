# Metadata and Userdata handling

Most providers offer a mechanism to provide a OS with some additional
metadata as well as custom userdata. `Metadata` in this context is
fixed information provided by the provider (e.g. the host
name). `Userdata` is completely custom data which a user can supply to
the instance.

The [metadata package](../pkg/metadata/) handles both metadata and
userdata for a number of providers (see below).  It abstracts over the
provider differences by exposing both metadata and userdata in a
directory hierarchy under `/var/config`.  For example, sshd config
files from the metadata are placed under `/var/config/ssh`.

Userdata is assumed to be a single string and the contents will be
stored under `/var/config/userdata`.  If userdata is a json file, the
contents will be further processed, where different keys cause
directories to be created and the directories are populated with files. Foer example, the following userdata file:
```
{
    "ssh" : {
        "sshd_config" : {
            "perm" : "0600",
            "content": "PermitRootLogin yes\nPasswordAuthentication no"
        }
    },
    "foo" : {
        "bar" : {
            "perm": "0644",
            "content": "foobar"
        },
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

This hierarchy can then be used by individual containers, who can bind
mount the config sub-directory into their namespace where it is
needed.


# Providers

Below is a list of supported providers and notes on what is supported. We will add more over time.


## GCP

GCP metadata is reached via a well known URL
(`http://metadata.google.internal/`) and currently
we extract the hostname and populate the
`/var/config/ssh/authorized_keys` from metadata. In the future we'll
add more complete SSH support.

GCP userdata is extracted from `/computeMetadata/v1/instance/attributes/userdata`.


## HyperKit

HyperKit does not support metadata and userdata is passed in as a single file via a ISO9660 image.

