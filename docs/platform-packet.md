# LinuxKit with bare metal on Packet

Packet is a bare metal hosting provider.

You will need to have created a Packet account and a project to
put this new machine into. You will also need to create an API key
with appropriate permissions to 

Linuxkit on Packet boots via iPXE. This requires that you have
an HTTP server on which you can store your images. At the moment
there is no equivalent to "linuxkit push" to upload these images,
so you will have to host them yourself.

## Boot

`linuxkit run packet -api-key PACKET_API_KEY -base-url http://path-to-my-pxe-boot-server ...`

Servers take several minutes to provision.

## Console

Prepare an SSH public key?

## Disks

Elastic storage, pointer to.

## Networking

Make sure that the interfaces come up bonded?

## Integration services and Metadata
