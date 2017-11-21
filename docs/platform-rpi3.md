# LinuxKit on a Raspberry Pi 3b

LinuxKit supports building and booting images on a Raspberry Pi 3b
using the mainline arm64 bit kernels. The LinuxKit arm64 kernel has
support for some of the devices on the Raspberry Pi 3b, but notably it
does *not* support:

- WLAN
- Bluetooth
- Graphics (though text console should work)

It is unlikely, that we going to add support for these in the main
LinuxKit kernels in the near future. The LinuxKit kernel is more
targeted at VMs and baremetal servers where support for these type of
devices is typically not needed. However, it should be possible to
easily extend the LinuxKit kernel build process add the required
kernel options, in a similar fashion to how `-dbg` kernels are
build. See the [`kernel`](./kernels.md) documentation for details.


## Boot

We use the mainline Linux kernels for the Raspberry Pi and it is
booted via [`uboot`](https://www.denx.de/wiki/U-Boot). The `moby`
tool, via `linuxkit build -format rpi3 <YAML>`, currently produces a `tar`
archive which can be extracted onto a FAT32 formatted SD card to boot
your Raspberry Pi.

Currently, the root filesystem is provided as a RAM disk via the
`initrd`.


## Console

The LinuxKit images support console access via HDMI and USB keyboard
as well as via serial. For serial console, you need a suitable cable
to connect to the GPIO pins as described
[here](https://elinux.org/RPi_Serial_Connection).


## Disks

There currently is no support for persistent disks for the Raspberry
Pi. It may be possible to partition the SD card, format a second
partition as `ext4` (or similar), and use it for persistent storage.

**TODO:** Experiment with and document this set up.


## Networking

The onboard, USB connected network interface is supported by the
LinuxKit kernel, but for some unknown reason the driver is not cold
plugged by `mdev`. To use the network interface, the driver needs to
be `modprobe`d before the network interface can be used. The easiest
way is to add the following section to the `onboot` section in your
LinuxKit YAML file:

```
  - name: netdev
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "smsc95xx"]
```

**TODO:** Figure out why mdev is not loading the driver.
