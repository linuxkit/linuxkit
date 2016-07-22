## List of platforms

These are the supported platforms. Each should boot with `mobyplatform=xxx` in the command line.

### Desktop
`mac` `windows` https://github.com/docker/pinata

The desktop editions.

### Cloud
`aws` `azure` https://github.com/docker/editions

The cloud editions

### Test
`test` https://github.com/docker/moby

Internal test target

### Running on other platforms
`unknown`

The default fallback target name is `unknown` that does only default setup.

## Notes on other platforms

KVM/machine: https://github.com/docker/moby/pull/225

Xen PV: a suitable config is
```
name = "moby0"
memory = 1024
vcpus=1
disk = ['phy:/dev/vg0/moby0,hda,w']
vif = [ 'mac=00:22:ab:42:99:00, bridge=br0' ]
kernel="/home/justin/images/moby/vmlinuz64"
ramdisk="/home/justin/images/moby/initrd.img.gz"
extra="console=hvc0"
on_reboot="restart"
```

VMWare Fusion: Should work fine, IDE, SATA or SCSI disks will work, and the default network driver.
