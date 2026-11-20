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

## Provider detection

Without a provider list, metadata first reads local DMI/SMBIOS identifiers and the hypervisor UUID.
If these identify a supported cloud, it probes only that cloud's provider.
If the probe fails, it logs an error and tries CDROM data without probing other network providers.
Azure, Oracle Cloud and Alibaba Cloud are recognised but have no metadata provider; they skip network probes and go straight to CDROM.

Unknown hosts, including VMware, plain QEMU/KVM and systems without DMI, probe providers in parallel with a shared 30-second deadline.
Network providers rank in this order: aws/openstack, gcp, hetzner, scaleway, vultr, digitalocean, metaldata.
A successful network probe wins once all higher-ranked probes have failed and VMware guestinfo has finished without data.
Other probes are cancelled, and any other providers that have already answered are logged as conflicts at error level.
Populated VMware guestinfo takes priority over every network provider; see [VMware](#vmware).
If no probe succeeds, metadata tries CDROM data.

AWS and OpenStack share one probe of `/latest/meta-data/hostname` at `169.254.169.254`, using the AWS token flow.
After that succeeds, a successful request to `/openstack/latest/meta_data.json` selects OpenStack.
An HTTP error from that request selects AWS, preserving support for other EC2-compatible services.
A network error is retried until the deadline and does not identify the service as AWS.

To select providers explicitly, set the metadata container's command, for example `command: ["/usr/bin/metadata", "gcp", "cdrom"]`.
The list overrides DMI selection.
If DMI identifies a supported cloud whose provider is absent from the list, metadata logs a warning and uses the list.
Unknown or recognised-but-unsupported clouds do not cause a mismatch warning.
Explicit lists retain their existing precedence: network providers first, then local VMware/CDROM providers, then `file=PATH` providers, preserving list order within each group.
These probes run sequentially, each with its own 30-second budget.
Only providers named in the list are tried.

Every metadata HTTP request retries network errors, such as refused connections, resets and timeouts, with delays of 1, 2, 4, then 5 seconds between attempts until its deadline.
Each request has a two-second timeout; HTTP errors are final and are not retried.
These rules also apply to metadata extraction, EC2 disambiguation and Scaleway's privileged-port userdata requests.
Parallel detection shares one 30-second budget; extraction gets a fresh 30-second budget for all its requests.
An unknown host without a metadata service can therefore wait about 30 seconds before trying CDROM.
Use an explicit `cdrom` or `file=PATH` list to skip network probing.

Before setting the kernel hostname, metadata trims whitespace.
A hostname longer than 64 bytes is reduced to its first DNS label, cut to 63 bytes if that label is still too long.
This adjustment is logged and leaves `/run/config/hostname` unchanged.
GCP always writes the short hostname to that file, so readers that previously received a GCP FQDN now receive its first label.

# Providers

The following sections describe provider-specific metadata and userdata handling.


## GCP

GCP metadata is reached at `http://metadata.google.internal/computeMetadata/v1/`.
The provider writes the first label of `instance/hostname` to `/run/config/hostname` and populates `/run/config/ssh/authorized_keys` from metadata.

GCP userdata is extracted from `/computeMetadata/v1/instance/attributes/user-data` and made available in `/run/config/userdata`.

## AWS

AWS metadata is reached at `http://169.254.169.254/latest/meta-data/`.
The provider extracts the hostname and populates `/run/config/ssh/authorized_keys` from metadata.

AWS userdata is extracted from `http://169.254.169.254/latest/user-data` and made available in `/run/config/userdata`.

Requests use IMDSv2 session tokens.
If the token request gets an HTTP error, the provider falls back to IMDSv1.
Network errors follow the shared [retry and deadline rules](#provider-detection).

## Hetzner

Hetzner metadata is reached at `http://169.254.169.254/hetzner/v1/metadata/`.
The provider reads `hostname`, `instance-id`, `public-ipv4` and `public-keys`, writing the hostname, instance ID, public IPv4 address and SSH authorized keys under `/run/config`.
It no longer uses the EC2-compatible `/latest/` routes and no longer writes `/run/config/local_ipv4`.

Hetzner userdata is extracted from `http://169.254.169.254/hetzner/v1/userdata` and made available in `/run/config/userdata`.

## VMware

VMware guestinfo is supported on Linux 386 and amd64 only.
On unknown hosts, populated `guestinfo.metadata` or `guestinfo.userdata` wins immediately over network providers and cancels their probes.
A network provider that has already answered is logged as a conflict.
This changes the previous network-first selection order when both sources have data.
On other architectures, metadata uses network providers and then CDROM; explicit lists can also use `file=PATH`.

## HyperKit

HyperKit does not distinguish metadata and userdata, it's simply
refered to as data, which is passed to the VM as a disk image
in ISO9660 format.

## Virtualization.Framework

Virtualization.Framework does not distinguish metadata and userdata, it's simply
refered to as data, which is passed to the VM as a disk image
in ISO9660 format.
