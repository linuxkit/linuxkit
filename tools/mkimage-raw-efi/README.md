# ESP Structure Overview

The script initializes an EFI System Partition using `mkfs.vfat` and populates it with directories and files for **systemd-boot** and a Linux Unified Kernel Image (UKI).

## Partition Layout

```bash
ESP
├── EFI
│   ├── BOOT # contains exactly one of the bootloader binaries below for the respective architecture
│   │   ├── BOOTX64.EFI # amd64
│   │   ├── BOOTAA64.EFI # arm64
│   │   └── BOOTRISCV64.EFI # riscv64
│   └── Linux
│       └── linuxkit.efi # LinuxKit Unified Kernel Image (UKI)
└── loader
    └── loader.conf # systemd-boot configuration file
```

UKIs in `EFI/Linux` do not need an explicit entry in `loader/entries` but are automatically picked up by `systemd-boot`.
