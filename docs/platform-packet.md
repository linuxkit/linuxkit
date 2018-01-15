# LinuxKit with bare metal on Packet

[Packet](http://packet.net) is a bare metal hosting provider.

You will need to [create a Packet account] and a project to
put this new machine into. You will also need to [create an API key]
with appropriate read/write permissions to allow the image to boot.

[create a Packet account]:https://app.packet.net/#/registration/
[create an API key]:https://help.packet.net/quick-start/api-integrations

Linuxkit is known to boot on the [Type 0] 
and [Type 1] servers at Packet.
Support for other server types, including the [Type 2A] ARM server,
is a work in progress.

[Type 0]:https://www.packet.net/bare-metal/servers/type-0/
[Type 1]:https://www.packet.net/bare-metal/servers/type-1/
[Type 2A]:https://www.packet.net/bare-metal/servers/type-2a/

The `linuxkit run packet` command can mostly either be configured via
command line options or with environment variables. see `linuxkit run
packet --help` for the options and environment variables.

By default, `linuxkit run` will provision a new machine and remove it
once you are done. With the `-keep` option the provisioned machine
will not be removed. You can then use the `-device` option with the
device ID on subsequent `linuxkit run` invocations to re-use an
existing machine. These subsequent runs will update the iPXE data so
you can boot alternative kernels on an existing machine.

There is an example YAML file for [x86_64](../examples/packet.yml) and
an additional YAML for [arm64](../examples/packet.arm64.yml) servers
which provide both access to the serial console and via ssh and
configures bonding for network devices via metadata (if supported).

For x86_64 builds for Intel servers we strongly recommend adding
`ucode: intel-ucode.cpio` to the kernel section in the YAML. This
updates the Intel CPU microcode to the latest by prepending it to the
generated initrd file. The `ucode` entry is only recommended when
booting on baremetal. It should be omitted (but is harmless) when
building images to boot in VMs.

**Note**: The update of the iPXE configuration sometimes may take some
time and the first boot may fail. Hitting return on the console to
retry the boot typically fixes this.

## Boot

LinuxKit on Packet boots the `kernel+initrd` output from moby via
[iPXE](https://help.packet.net/technical/infrastructure/custom-ipxe)
which also requires a iPXE script. iPXE booting requires a HTTP server
on which you can store your images. The `-base-url` option specifies
the URL to a HTTP server from which `<name>-kernel`,
`<name>-initrd.img`, and `<name>-packet.ipxe` can be downloaded during
boot.

If you have your own HTTP server, you can use `linuxkit push packet`
to create the files (including the iPXE script) you need to make
available.

If you don't have a public HTTP server at hand, you can use the
`-serve` option. This will create a local HTTP server which can either
be run on another Packet machine or be made accessible with tools
like [ngrok](https://ngrok.com/).

For example, to boot the [example](../examples/packet.net)
with a local HTTP server:

```sh
linuxkit build packet.yml
# run the web server
# run 'ngrok http 8080' in another window
PACKET_API_KEY=<API key> PACKET_PROJECT_ID=<Project ID> \
linuxkit run packet -serve :8080 -base-url <ngrok url> packet
```

To boot a `arm64` image for Type 2a machine (`-machine baremetal_2a`)
you currently need to build using `linuxkit build packet.yml
packet.arm64.yml` and then un-compress both the kernel and the initrd
before booting, e.g:

```sh
mv packet-initrd.img packet-initrd.img.gz && gzip -d packet-initrd.img.gz
mv packet-kernel packet-kernel.gz && gzip -d packet-kernel.gz
```

The LinuxKit image can then be booted with:

```sh
PACKET_API_KEY=<API key> PACKET_PROJECT_ID=<Project ID> \
linuxkit run packet -machine baremetal_2a  -serve :8080 -base-url -base-url <ngrok url> packet
```

Alternatively, `linuxkit push packet` will uncompress the kernel and
initrd images on arm machines (or explicitly via the `-decompress`
flag. There is also a `linuxkit serve` command which will start a
local HTTP server serving the specified directory.

**Note**: It may take several minutes to deploy a new server. If you
are attached to the console, you should see the BIOS and the boot
messages.


## Console

By default, `linuxkit run packet ...` will connect to the
Packet
[SOS ("Serial over SSH") console](https://help.packet.net/technical/networking/sos-rescue-mode). This
requires `ssh` access, i.e., you must have uploaded your SSH keys to
Packet beforehand.

You can exit the console vi `~.` on a new line once you are
disconnected from the serial, e.g. after poweroff.

**Note**: We also require that the Packet SOS host is in your
`known_hosts` file, otherwise the connection to the console will
fail. There is a Packet SOS host per zone.

You can disable the serial console access with the `-console=false`
command line option.


## Disks

At this moment the Linuxkit server boots from RAM, with no persistent
storage.  We are working on adding persistent storage support on Packet.


## Networking

On the baremetal type 2a system (arm64 Cavium Thunder X) the network device driver does not get autoloaded by `mdev`. Please add:

```
  - name: modprobe
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "nicvf"]
```

to your YAML files before any containers requiring the network to be up, e.g., the `dhcpcd` container.

Some Packet server types have bonded networks; the `metadata` package has support for setting
these up, and also for adding additional IP addresses.


## Integration services and Metadata

Packet supports [user state](https://help.packet.net/technical/infrastructure/user-state)
during system bringup, which enables the boot process to be more informative about the
current state of the boot process once the kernel has loaded but before the
system is ready for login.
