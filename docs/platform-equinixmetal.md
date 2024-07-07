# LinuxKit with bare metal on Equinix Metal

[Equinix Metal](http://deploy.equinix.com) is a bare metal hosting provider.

You will need to [create an Equinix Metal account] and a project to
put this new machine into. You will also need to [create an API key]
with appropriate read/write permissions to allow the image to boot.

[create an Equinix Metal account]:https://console.equinix.com/sign-up
[create an API key]:https://deploy.equinix.com/developers/docs/metal/identity-access-management/api-keys/

The `linuxkit run equinixmetal` command can mostly either be configured via
command line options or with environment variables. see `linuxkit run
equinixmetal --help` for the options and environment variables.

By default, `linuxkit run` will provision a new machine and remove it
once you are done. With the `-keep` option the provisioned machine
will not be removed. You can then use the `-device` option with the
device ID on subsequent `linuxkit run` invocations to re-use an
existing machine. These subsequent runs will update the iPXE data so
you can boot alternative kernels on an existing machine.

There is an example YAML file for [x86_64](../examples/equinixmetal.yml) and
an additional YAML for [arm64](../examples/equinixmetal.arm64.yml) servers
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

LinuxKit on Equinix Metal boots the `kernel+initrd` output from moby via
[iPXE](https://deploy.equinix.com/developers/docs/metal/operating-systems/custom-ipxe/)
which also requires a iPXE script. iPXE booting requires a HTTP server
on which you can store your images. The `-base-url` option specifies
the URL to a HTTP server from which `<name>-kernel`,
`<name>-initrd.img`, and `<name>-equinixmetal.ipxe` can be downloaded during
boot.

If you have your own HTTP server, you can use `linuxkit push equinixmetal`
to create the files (including the iPXE script) you need to make
available.

If you don't have a public HTTP server at hand, you can use the
`-serve` option. This will create a local HTTP server which can either
be run on another Equinix Metal machine or be made accessible with tools
like [ngrok](https://ngrok.com/).

For example, to boot the [example](../examples/platform-equinixmetal.yml)
with a local HTTP server:

```sh
linuxkit build platform-equinixmetal.yml
# run the web server
# run 'ngrok http 8080' in another window
METAL_AUTH_TOKEN=<API key> METAL_PROJECT_ID=<Project ID> \
linuxkit run equinixmetal -serve :8080 -base-url <ngrok url> equinixmetal
```

To boot a `arm64` image for Type 2a machine (`-machine baremetal_2a`)
you currently need to build using `linuxkit build equinixmetal.yml
equinixmetal.arm64.yml` and then un-compress both the kernel and the initrd
before booting, e.g:

```sh
mv equinixmetal-initrd.img equinixmetal-initrd.img.gz && gzip -d equinixmetal-initrd.img.gz
mv equinixmetal-kernel equinixmetal-kernel.gz && gzip -d equinixmetal-kernel.gz
```

The LinuxKit image can then be booted with:

```sh
METAL_API_TOKEN=<API key> METAL_PROJECT_ID=<Project ID> \
linuxkit run equinixmetal -machine baremetal_2a  -serve :8080 -base-url -base-url <ngrok url> equinixmetal
```

Alternatively, `linuxkit push equinixmetal` will uncompress the kernel and
initrd images on arm machines (or explicitly via the `-decompress`
flag. There is also a `linuxkit serve` command which will start a
local HTTP server serving the specified directory.

**Note**: It may take several minutes to deploy a new server. If you
are attached to the console, you should see the BIOS and the boot
messages.


## Console

By default, `linuxkit run equinixmetal ...` will connect to the
Equinix Metal
[SOS ("Serial over SSH") console](https://deploy.equinix.com/developers/docs/metal/resilience-recovery/serial-over-ssh/). This
requires `ssh` access, i.e., you must have uploaded your SSH keys to
Equinix Metal beforehand.

You can exit the console vi `~.` on a new line once you are
disconnected from the serial, e.g. after poweroff.

**Note**: We also require that the Equinix Metal SOS host is in your
`known_hosts` file, otherwise the connection to the console will
fail. There is a Equinix Metal SOS host per zone.

You can disable the serial console access with the `-console=false`
command line option.


## Disks

At this moment the Linuxkit server boots from RAM, with no persistent
storage.  We are working on adding persistent storage support on Equinix Metal.


## Networking

On the baremetal type 2a system (arm64 Cavium Thunder X) the network device driver does not get autoloaded by `mdev`. Please add:

```
  - name: modprobe
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "nicvf"]
```

to your YAML files before any containers requiring the network to be up, e.g., the `dhcpcd` container.

Some Equinix Metal server types have bonded networks; the `metadata` package has support for setting
these up, and also for adding additional IP addresses.


## Integration services and Metadata

Equinix Metal supports [user state](https://deploy.equinix.com/developers/docs/metal/server-metadata/user-data/)
during system bringup, which enables the boot process to be more informative about the
current state of the boot process once the kernel has loaded but before the
system is ready for login.
