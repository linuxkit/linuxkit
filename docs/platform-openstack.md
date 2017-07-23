# LinuxKit with OpenStack

LinuxKit interacts with OpenStack through its native APIs and requires access
to both an OpenStack Keystone server for authentication and an OpenStack image
service (Glance) in order to host the LinuxKit images.

Supported (tested) versions of the relevant OpenStack APIs:

- Keystone v3
- Glance v2

## Push

### Image types supported:
- **ami** (Amazon Machine image)
- **vhd** (Hyper-V)
- **vhdx** (Hyper-V)
- **vmdk** (VMware Disk)
- **raw** (Raw disk image)
- **qcow2** (Qemu disk image)
- **iso** (ISO9660 compatible CD-ROM image)

A compatible image needs to have the correct extension (must match
one from above) in order to be supported by the `linuxkit push openstack` 
command. The `openstack` backend will use the filename extension to determine
the image type, and use the filename as a label for the new image. 

### Usage

The `openstack` backend uses the password authentication method in order to
retrieve a token that can be used to interact with the various components of
OpenStack.  Example usage:

```
./linuxkit push openstack \
-authurl=http://keystone.com:5000/v3 \
-username=admin \
-password=XXXXXXXXXXX \
-project=linuxkit \
./linuxkit.iso 
```

### Execution Flow
1. Authenticate with OpenStack's identity service
2. Create a "queued" image in Glance and retrieve its UUID
3. Use this new image ID to upload the LinuxKit image
