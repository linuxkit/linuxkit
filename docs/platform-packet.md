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

## Boot

Build an image with `moby build`. The [packet.yml](https://github.com/vielmetti/linuxkit/blob/master/examples/packet.yml)
example file provides a suitable template to start from.

Linuxkit on Packet [boots via iPXE]. This requires that you have
an HTTP server on which you can store your images. At the moment
there is no equivalent to "linuxkit push" to upload these images,
so you will have to host them yourself. The images can be served
from any HTTP server, though in the interest of performance you may
want to locate those images near the data center that you're booting in.

[boots via iPXE]:https://help.packet.net/technical/infrastructure/custom-ipxe

Servers take several minutes to provision. During this time their
state can be seen from the Packet console.

```
$ linuxkit run packet --help
USAGE: linuxkit run packet [options] [name]

Options:

  -api-key string
    	Packet API key
  -base-url string
    	Base URL that the kernel and initrd are served from.
  -hostname string
    	Hostname of new instance (default "moby")
  -img-name string
    	Overrides the prefix used to identify the files. Defaults to [name]
  -machine string
    	Packet Machine Type (default "baremetal_0")
  -project-id string
    	Packet Project ID
  -zone string
    	Packet Zone (default "ams1")
 ```
## Console

If your LinuxKit system does not include an ssh or remote console 
application, you can still connect to it via the Packet SOS ("Serial over SSH")
console. See https://help.packet.net/technical/networking/sos-rescue-mode
for details on that mode.

## Disks

At this moment the Linuxkit server boots from RAM, with no persistent
storage and there is no code that mounts disks. As a result,
when the Linuxkit image reboots, all is lost. 

Packet supports a [persistent iPXE] mode through its API
which would allow a server to come back up after a reboot
and re-start the PXE process. This is great for testing your
provisioning scripts. This is not yet available directly
through Linuxkit.

[persistent iPXE]:https://help.packet.net/technical/infrastructure/custom-ipxe

## Networking

Some Packet server types have bonded networks; the current code does
not support that.

## Integration services and Metadata

Packet supports [user state](https://help.packet.net/technical/infrastructure/user-state)
during system bringup, which enables the boot process to be more informative about the
current state of the boot process once the kernel has loaded but before the
system is ready for login.
