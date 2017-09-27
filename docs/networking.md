# Networking

While the common pattern of using LinuxKit inside the virtualized environment
doesn't usually bring any network connectivity issues, they still may occur.
Moreover, LinuxKit may be used on the real hardware, and in that case the
network setup could be a major part of the getting the image run.

## General Steps

There is a standard checklist one have to follow to set up the networking:

1. Drivers are loaded by kernel (otherwise, may need a [custom kernel](kernels.md));
2. Firmware is loaded (otherwise, need to put firmware blobs into the image);
3. Network interfaces are visible in the system (otherwise, need to revisit items 1 and 2);
4. Wi-Fi: wpa_supplicant is running (otherwise, need to include it into services section);
5. DHCP: dhcpcd service is loaded.
6. May need to set up static IP addressed and routes, but it's out of scope of this document.

Usually, `lspci` provides the information about the detected devices, and
`dmesg` prints the information whether the drivers for those devices
are loaded or not (usually if nothing mentioned in the log, then no driver is loaded).

### Wi-Fi Notes

Vanilla LinuxKit kernel doesn't provide the Wi-Fi support, therefore
the custom kernel should be used. The major options to enable:
 - `CONFIG_WIRELESS=y` (Wireless);
 - `CONFIG_CFG80211=y` (Improved wireless configuration API);
 - `CONFIG_MAC80211=y` (Generic IEEE 802.11 Networking Stack);
 - `CONFIG_WLAN=y` (Wireless LAN).

`wpa_supplicant` requires CONFIG_CFG80211 to configure the Wi-Fi connections.
CONFIG_CFG80211_WEXT may be useful too for the old hardware.

The particular driver is better to compile as a module (and it's mandatory if it
requires a userspace firmware), and enable it with the modprobe later.
Please note that enabling the driver family is not always enough, for example,
Broadcom FullMAC device (`CONFIG_BRCMFMAC=m`) may need to have PCIE bus interface
support (`CONFIG_BRCMFMAC_PCIE=y`) enabled too to work.

In the image configuration please add the modprobe item
(with the example of the Broadcom FullMAC device):
```
onboot:
  - name: modprobe
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "-a", "brcmfmac"]
```

The next step is getting an optional firmware. Please use the official
[Linux Wireless wiki](https://wireless.wiki.kernel.org/welcome) and search for the
driver instructions there (`dmesg` may give the helpful messages too).

### Userspace firmware

If the driver requires a firmware, please download it from the official
[Repository of firmware blobs](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git/tree/). You can either clone the repository, or just download a particular blob (using "(plain)" link).

Please use the files section to put the blobs into the image
(with the example of the BCM43602 device mentioned above,
and the assumption that `brcmfmac43602-pcie.bin` is in the current directory):
```
files:
  - path: /lib/firmware/brcm/brcmfmac43602-pcie.bin
    source: brcmfmac43602-pcie.bin
```

Please note that `dmesg` may still give a warning that firmware is missing, but
it can be ignored. Please use `lsmod` to verify that the module is actually loaded.
Also you could see the network interfaces available using `ifconfig -a`.

### WPA Supplicant

The supplicant is required for connecting to a password protected wireless access point.
Please note that it will set the network interface up automatically, but you may need
to wait some time until DHCP address is assigned (if you use DHCP).

Please update the image configuration:
```
services:
  - name: wpa_supplicant
    image: linuxkit/wpa_supplicant:<hash>
    binds:
     - /etc/wpa_supplicant:/etc/wpa_supplicant
    command: ["/sbin/wpa_supplicant", "-i", "wlan0", "-c", "/etc/wpa_supplicant/wpa_supplicant.conf"]
files:
  - path: etc/wpa_supplicant/wpa_supplicant.conf
    contents: |
      network={
        ssid="<ssid>"
        psk="<password>"
      }
```

Please be aware that "-B" option (as it recommended all over the Internet) is not used,
because it runs as a service already (moreover, with this option the container
stops immediately and Wi-Fi will not work).

### DHCP Client

The original configuration for LinuxKit recommends to run `dhcpcd` on boot, but
it won't work for the Wi-Fi devices (because `wpa_supplicant` can't
be used on boot due to its requirement to be always running).

Please update the image configuration if you're using Wi-Fi:
```
services:
  - name: dhcpcd
    image: linuxkit/dhcpcd:<hash>
    command: ["/sbin/dhcpcd", "wlan0"]
```

You may want to use a configuration file too - the sample above provides just basic functionality.

## Sample linuxkit.yml

At the end, you will have `linuxkit.yml` like this for the Wi-Fi configuration:
```
kernel:
  image: <custom kernel>
  cmdline: "console=tty0"
init:
  - linuxkit/init:<hash>
  - linuxkit/runc:<hash>
  - linuxkit/containerd:<hash>
onboot:
  - name: modprobe
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "<driver>"]
services:
  - name: wpa_supplicant
    image: linuxkit/wpa_supplicant:<hash>
    binds:
     - /etc/wpa_supplicant:/etc/wpa_supplicant
    command: ["/sbin/wpa_supplicant", "-i", "wlan0", "-c", "/etc/wpa_supplicant/wpa_supplicant.conf"]
  - name: dhcpcd
    image: linuxkit/dhcpcd:<hash>
    command: ["/sbin/dhcpcd", "wlan0"]
  - name: getty
    image: linuxkit/getty:<hash>
    env:
     - INSECURE=true
files:
files:
  - path: /lib/firmware/<firmware>
    source: <firmware>
  - path: etc/wpa_supplicant/wpa_supplicant.conf
    contents: |
      network={
        ssid="<ssid>"
        psk="<password>"
      }
trust:
  org:
    - linuxkit
```
