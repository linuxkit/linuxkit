# LinuxKit with bare metal on Packet

Packet is a bare metal hosting provider.

You will need to have created a Packet account and a project to
put this new machine into. You will also need to create an API key
with appropriate read/write permissions to allow the image to boot.

Linuxkit on Packet boots via iPXE. This requires that you have
an HTTP server on which you can store your images. At the moment
there is no equivalent to "linuxkit push" to upload these images,
so you will have to host them yourself. The images can be served
from any HTTP server, though in the interest of performance you'll
want to locate those images near the data center that you're booting in.

Linuxkit is known to work on the Type 0 server at Packet.

## Boot

`linuxkit run packet -api-key PACKET_API_KEY -base-url http://path-to-my-pxe-boot-server ...`

Servers take several minutes to provision. During this time their
state can be seen from the Packet console.

## Console

If your LinuxKit system does not include an ssh or remote console 
application, you can still connect to it via the Packet SOS ("Serial over SSH")
console. See https://help.packet.net/technical/networking/sos-rescue-mode
for details on that mode.

## Disks

At this moment the Linuxkit server boots from RAM, with no persistent
storage. 

## Networking

Make sure that the interfaces come up bonded?

## Integration services and Metadata
