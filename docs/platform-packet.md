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

## Boot



LinuxKit on Packet boots the `kernel+initrd` output from moby
via
[iPXE](https://help.packet.net/technical/infrastructure/custom-ipxe). iPXE
booting requires a HTTP server on which you can store your images. At
the moment there is no builtin support for this, although we are
working on this too.

A simple way to host files is via a simple local http server written in Go, e.g.:

```Go
package main

import (
	"log"
	"net/http"
)

func main() {
	// Simple static webserver:
	log.Fatal(http.ListenAndServe(":8080", http.FileServer(http.Dir("."))))
}
```

and then `go run` this in the directory with the `kernel+initrd`.

The web server must be accessible from Packet. You can either run the
server on another Packet machine, or use tools
like [ngrok](https://ngrok.com/) to make it accessible via a reverse
proxy.

You then specify the location of your http server using the
`-base-url` command line option. For example, to boot the
toplevel [linuxkit.yml](../linuxkit.yml) example:

```sh
moby build linuxkit.yml
# run the web server
# run 'ngrok http 8080' in another window
PACKET_API_KEY=<API key> linuxkit run packet -base-url http://9b828514.ngrok.io -project-id <Project ID> linuxkit
```

**Note**: It may take several minutes to deploy a new server. If you
are attached to the console, you should see the BIOS and the boot
messages.


## Console

By default, `linuxkit run packet ...` will connect to the
Packet
[SOS ("Serial over SSH") console](https://help.packet.net/technical/networking/sos-rescue-mode). This
requires `ssh` access, i.e., you must have uploaded your SSH keys to
Packet beforehand.

**Note**: We also require that the Packet SOS host is in your
`known_hosts` file, otherwise the connection to the console will
fail. There is a Packet SOS host per zone.

You can disable the serial console access with the `-console=false`
command line option.


## Disks

At this moment the Linuxkit server boots from RAM, with no persistent
storage.  We are working on adding persistent storage support on Packet.


## Networking

Some Packet server types have bonded networks; the current code does
not support that.

## Integration services and Metadata

Packet supports [user state](https://help.packet.net/technical/infrastructure/user-state)
during system bringup, which enables the boot process to be more informative about the
current state of the boot process once the kernel has loaded but before the
system is ready for login.
