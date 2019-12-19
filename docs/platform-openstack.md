# LinuxKit with OpenStack

LinuxKit interacts with OpenStack through its native APIs, providing basic support for pushing images and launching virtual instances.

Supported (tested) versions of the relevant OpenStack APIs are:

- Keystone v3
- Glance v2
- Nova v2
- Neutron v2

## Authentication

LinuxKit's support for OpenStack includes configuring access to your cloud as detailed in the official [os-client-config](https://docs.openstack.org/os-client-config/latest/user/configuration.html) documentation.

## Push

### Image types supported:
- **ami** (Amazon Machine image)
- **vhd** (Hyper-V)
- **vhdx** (Hyper-V)
- **vmdk** (VMware Disk)
- **raw** (Raw disk image)
- **qcow2** (Qemu disk image)
- **iso** (ISO9660 compatible CD-ROM image)

A compatible image needs to have the correct extension (must match one from above) in order to be supported by the `linuxkit push openstack` command. The `openstack` backend will use the filename extension to determine the image type, and use the filename as a label for the new image.

Images generated with Moby can be uploaded into OpenStack's image service with `linuxkit push openstack`, plus a few options.  For example:

```shell
./linuxkit push openstack \
  -img-name=LinuxKitTest
  ./linuxkit.iso
```

## Run

Virtual machines can be launched using `linuxkit run openstack`.  As an example:

```shell
linuxkit run openstack \
  -flavor=hotdog
  -keyname=deadline_ed25519 \
  -sec-groups=allow_ssh,nginx \
  -network c5d02c5f-c625-4539-8aed-1dab3aa85a0a \
  LinuxKitTest
```

This will create a new instance with the same name as the image, and if successful will return the newly-created instance's UUID.  You can then check the boot logs as follows, e.g:

```shell
$ openstack console log show 7cdd4d53-78b3-47c7-9a77-ba8a3f60548d
[..]
linuxkit-fa163ec840c9 login: root (automatic login)

Welcome to LinuxKit!

NOTE: This system is namespaced.
The namespace you are currently in may not be the root.
[..]
```
