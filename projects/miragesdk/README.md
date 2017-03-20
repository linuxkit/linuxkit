### Mirage SDK

Instructions:

```
../../bin/moby examples/mirage-dhcp.yaml`
../../scripts/qemu.sh mirage-dhcp-initrd.img mirage-dhcp-bzImage "$(bin/moby --cmdline mirage-dhcp.yaml)"
```
