# LinuxKit with OpenStack

LinuxKit interacts with OpenStack through its native APIs and requires access
to both an OpenStack Keystone server for authentication and a OpenStack Glance
server in order to host the LinuxKit images.

Supported (Tested) Versions:

- OpenStack Ocata Release
- Keystone v3 API 
- Glance v2 API

##Push

### Image types supported:
- **ami** (Amazon Machine image)
- **vhd** (Hyper-V)
- **vhdx** (Hyper-V)
- **vmdk** (VMware Disk)
- **raw** (Raw disk image)
- **qcow2** (Qemu disk image)
- **iso** (ISO9660 compatible CD-ROM image)

A compatible/supported image needs to have the correct extension (must match
one from above) in order to be supported by the `linuxkit push openstack` 
command. The `openstack` backend will use the filename extension to determine
the image type, and use the filename as a label for the new image. 

The `openstack` backend also supports OpenStack projects to provide
multi-tenancy support when uploading images. 

### Usage

The `openstack` backend uses the password authentication method in order to
retrieve a token that can be used to interact with the various components of
OpenStack. The URLs for the Keystone/Glance server components need to have 
the ports detailed as below.

```
./linuxkit push openstack \
-keystoneAddr=http://keystone.com:5000 \
-username=admin \
-password=XXXXXXXXXXX \
-project=linuxkit \
-glanceAddr=http://glance.com:9292 \
./linuxkit.iso 
```

### Execution Flow
1. Log in to OpenStack (Keystone)
2. Retrieve the OpenStack Key from the response header
3. Create a "queued" image on the glance server and return the new image ID
4. Use the new image ID and upload the LinuxKit image under this new ID
