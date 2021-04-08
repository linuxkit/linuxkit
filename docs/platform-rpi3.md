# LinuxKit on a Raspberry Pi 3b

LinuxKit supports building and booting images on a Raspberry Pi 3b
using the mainline arm64 bit kernels. The LinuxKit arm64 kernel has
support for some of the devices on the Raspberry Pi 3b and 4, but
notably it does *not* support:

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
tool, via `linuxkit build -format rpi <YAML>`, currently produces a `tar`
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

## SD Card

The built in SD Card is supported

For Raspberry Pi 4b use:
```yaml
  - name: block
    image: linuxkit/modprobe:v0.8
    command: ["modprobe", "mmc_block" ]
  - name: sdhci
    image: linuxkit/modprobe:v0.8
    command: ["modprobe", "sdhci-iproc" ]
```

## USB Sticks

To enable and external USB stick as disk, add the following to the
onboot section in your YAML:

```
  - name: usb-storage
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "usb_storage"]
```

## Networking

The onboard, USB connected network interface is supported by the
LinuxKit kernel, but for some unknown reason the driver is not cold
plugged by `mdev`. To use the network interface, the driver needs to
be `modprobe`d before the network interface can be used. The easiest
way is to add the following section to the `onboot` section in your
LinuxKit YAML file:

```yaml
  - name: netdev
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "smsc95xx"]
```

For Raspberry Pi 3b+ use:
```yaml
  - name: netdev
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "lan78xx"]
```

For Raspberry Pi 4b use:
```yaml
  - name: netdev
    image: linuxkit/modprobe:v0.8
    command: ["modprobe", "mdio-bcm-unimac" ]
  - name: netdev
    image: linuxkit/modprobe:v0.8
    command: ["modprobe", "bcm7xxx" ]
```

## Examples
- [Raspberry Pi 3b](https://github.com/linuxkit/linuxkit/blob/master/examples/rpi3.yml)
- [Raspberry Pi 4b](https://github.com/linuxkit/linuxkit/blob/master/examples/rpi4.yml)

**TODO:** Figure out why mdev is not loading the driver.
