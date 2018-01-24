# LinuxKit with qemu/kvm

The `qemu` backend is the most versatile `run` backend for
`linuxkit`. It can boot both `x86_64` and `arm64` images, runs on
macOS and Linux (and possibly Windows), and can boot most types of
output formats. On Linux, `kvm` acceleration is enabled by default if
available.


## Boot

By default `linuxkit run qemu` will boot with the host architecture
(`x86_64` on `x86_64` machines and `aarch64` on `arm64` systems). The
architecture can be specified with `-arch` and currently accepts
`x86_64` and `aarch64` as arguments.

`linuxkit run qemu` can boot in different types of images:

- `kernel+initrd`: This is the default mode of `linuxkit run qemu` [`x86_64`, `arm64`]
- `iso-bios`: `linuxkit run qemu -iso <path to iso>` [`x86_64`]
- `iso-efi`: `linuxkit run qemu -iso -uefi <path to iso>`. This looks in `/usr/share/ovmf/bios.bin` for the EFI firmware by default. Can be overwritten with `-fw`. [`x86_64`, `arm64`]
- `qcow-bios`: `linuxkit run qemu disk.qcow2` [`x86_64`]
- `raw-bios`:  `linuxkit run qemu disk.img` [`x86_64`]
- `aws`: `linuxkit run qemu disk.img` boots a raw AWS disk image. [`x86_64`]

The formats `qcow-efi` and `raw-efi` may also work, but are currently not tested.


## Console

With `linuxkit run qemu` the serial console is redirected to stdio,
providing interactive access to the VM. You can specify `-gui` to get
a console window.


## Disks

The qemu backend supports multiple disks to be attached to the VM
using the standard `linuxkit` `-disk` syntax. The qemu backend
supports a number of different disk formats.


## Networking

The `qemu` backend supports a number of networking options, depending
on the platform you are running. The default is the userspace
networking which provides the VM with a internal DHCP server and
network connectivity, but does not provide access to the VMs network
from the outside.

With user mode networking you can publish selected VM ports on the
host, using the `-publish` option. It uses the same syntax as the
`qemu` binary. For example `linuxkit run qemu -publish 8080:80
linuxkit` exposes port `80` from the VM as port `8080` on the host.

On Linux, you can attach the VM either to an existing bridge or tap
interface. These require root privileges and you may want to use the
[`qemu-bridge-helper`](http://wiki.qemu.org/Features/HelperNetworking). To
attach to an existing bridge `br0` (e.g., one created with
`virt-manager`) you can use `linuxkit run qemu -networking
bridge,br0 linuxkit`.


## Integration services and Metadata

The `qemu` backend also allows passing custom userdata into the
[metadata package](./metadata.md) using either the `-data` or
`-data-file` command-line option. This attaches a CD device with the
data on.

If the `linuxkit/qemu-ga` package is added to the YAML the [Qemu Guest
Agent](https://wiki.libvirt.org/page/Qemu_guest_agent) will be
enabled. This provides better integration with `libvirt`.
