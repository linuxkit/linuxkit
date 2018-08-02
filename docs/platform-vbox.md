# LinuxKit with VirtualBox

LinuxKit can run using Oracle VirtualBox. This should work on OSX, Linux
and Windows. The standard install should be sufficient.

NB: Windows support is not currently working but should be fixed soon.

## Boot

The Virtualbox backend currently supports booting from disks or ISOs.
It should work with either BIOS (default) or EFI
(with `linuxkit run vbox --uefi ...`).

## Console

With `linuxkit run vbox` the serial console is redirected to
stdio, providing interactive access to the VM.


## Disks

The Virtualbox backend support configuring a persistent disk using the
standard `linuxkit` `-disk` syntax. Multiple disks are
supported and can be created in `raw` format; other formats that VirtualBox
supports can be attached. Note that additional drives are attached to the
SATA Controller, unlike the VM disk which is on the IDE Controller.

## Networking

You can select the networking mode, which defaults to the standard `nat`, by using the
`-networking` command-line option. Some networking modes (`hostonly`, `bridge`) will require 
the additional `adapter` parameter to the `-networking` option:

~~~
-networking hostonly,adapter=vboxnet0
~~~

You can specify more than one `-networking` option to setup multiple adapters. It is
recommended to setup the first adapter as `nat`.
