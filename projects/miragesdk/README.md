### Mirage SDK

Instructions:

```
../../bin/moby examples/mirage-dhcp.yml`
../../scripts/qemu.sh mirage-dhcp-initrd.img mirage-dhcp-bzImage "$(bin/moby --cmdline mirage-dhcp.yml)"
```
