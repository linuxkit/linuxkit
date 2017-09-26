# LinuxKit wpa_supplicant
Image with Wi-Fi Protected Access client and IEEE 802.1X supplicant for a [linuxkit](https://github.com/linuxkit/linuxkit)-generated image.


## Usage
The sample configuration for your `linuxkit.yml` assuming that wlan0 is your Wi-Fi device
(please note that you need the Linux kernel with Wi-Fi drivers enabled):

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

