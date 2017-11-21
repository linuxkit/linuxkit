# LinuxKit with VMware Fusion and vCenter

LinuxKit interacts with VMware desktop products (Fusion/Workstation) through the
`vmrun` utility that is preinstalled with the desktop products. The interaction
for VMware vSphere (single hypervisor) or VMware vCenter (vSphere management
platform) is performed through the XML SOAP API using the govmomi SDK.

Links:

- [VMware govmomi](https://github.com/vmware/govmomi)
- [VMware vmrun](https://www.vmware.com/support/ws55/doc/ws_learning_cli_vmrun.html)


Supported (Tested) versions:

- VMware Fusion 8.0/8.5
- VMware vSphere 6.0/6.5
- VMware vCenter 6.0


## Run
### VMware Workstation/Fusion
The backend `vmware` currently supports the booting of a `.vmdk` file that is
created through the `linuxkit build -format vmdk` command and is typically called with
`linuxkit run vmware <args> ./path`.

The WS/Fusion backend will construct a config version 8 (Hardware version 12)
`.vmx` file from the arguments that are passed to the `run` backend and then
use the `vmrun` utility to start the virtual machine. 

### VMware vSphere/vCenter
The backend `vsphere` currently supports booting through an `iso` file that is
created through the `linuxkit build -o iso-bios` and is started with `linuxkit run
vcenter <args> ./path`.

The vSphere/vCenter backend requires a user to have `pushed` a linuxkit `iso` to
a datastore before attempting to issue the `run` command. The VMware GO SDK is
used to build a new Virtual Machine from the configuration that is passed, the
new VM is then registered to the host passed as part of the `run` arguments. 

The `waitForIP` requires the `powerOn` argument and will make linuxkit wait
until the VM has both powered on and the VMware guest tools have started, it
will then print the guest IP address to `stdout`. This requires the 
`open-vm-tools` container to be added to the linuxkit OS .yml otherwise the wait
will eventually timeout.

## Push
### VMware vSphere/vCenter
To push an `iso` to a remote VMware datastore:

```
linuxkit push vcenter \
-url=<https://username:password@VCaddress/sdk> \
-datastore=<datastore_name> \
-datacenter=<dc_name> [optional, only needed if multiple DCs] \
-folder=<folder_name> [optional, will create a folder from the image name] \
-path=<iso_path>
```
Alternatively most arguments can be passed as environment variables:

- `VCURL` - VMware vCenter URL (ensure /sdk is appended)
- `VCDATACENTER` - VMware vCenter DataCenter name, if more than one
- `VCDATASTORE` - Name of a Datastore on that DataCenter 

## Console

VMware makes use of its own KVM (keyboard/video/mouse) console. With the
WS/Fusion backend the console will be displayed as the VM starts to boot. With
the vSphere/vCenter backend, the virtual machine will need finding in the
management tools and connecting to its console. 

**NOTE:** When building an instance for the vSphere/vCenter backend only a
single `tty` is needed as a serial device isn't added to the VM, adding a `ttyS`
will result in a debug message printed to the console every few seconds. 

## Disks
### VMware Workstation/Fusion
Adding a `-disk` will call on the `vmware-diskmanager` utility to create a disk
of the set size and add this to the `.vmx` configuration before starting the new
Virtual Machine.

### VMware vSphere/vCenter
The `-persistentSize <size in MB>` will create a persistent disk through the
VMware SDK and add this disk to the VM configuration. This disk will be created
in the same folder as the newly created VM. 

## Networking
### VMware Workstation/Fusion
Networking is automated and an interface as auto generated and will use NAT to
create a provide network access.
### VMware vSphere/vCenter
The `-network` argument can specify the name of either a vSwitch or a
Distributed vSwitch. When the VM is created a new VMXNet3 device is created and
places on the designated virtual switch. 

## Integration services and Metadata
The `open-vm-tools` container can be added to provide additional functionality
within a VMware vSphere and vCenter environment.

## Design decisions
For the `vcenter` backend, the decision was made to use only an `iso` as the
medium for the linuxkit VM instead of a `vmdk` file. The basis for this is in
the limitations between a `vmdk` on local storage and A `vmdk` that is hosted on
an actual VMware datastore (VMFS filesystem). Creating a local vmdk can make use
of thin provisioning, however it can't then be transferred to a VMware datastore
without converting the disk to a "fat" format. This has the result of turning a
relatively small upload, to an upload of perhaps 1GB. 
