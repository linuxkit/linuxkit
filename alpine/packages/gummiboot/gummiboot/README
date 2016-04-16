gummiboot Simple UEFI boot manager

gummiboot executes EFI images. The default entry is selected by a configured
pattern (glob) or an on-screen menu.

gummiboot operates on the EFI System Partition (ESP) only. Configuration
file fragments, kernels, initrds, other EFI images need to reside on the
ESP. Linux kernels must be built with CONFIG_EFI_STUB to be able to be
directly executed as an EFI image.

gummiboot reads simple and entirely generic configurion files; one file
per boot entry to select from.

Pressing Space (or most other) keys during bootup will show an on-screen
menu with all configured entries to select from. Pressing enter on the
selected entry loads and starts the EFI image.

If no timeout is configured and no key pressed during bootup, the default
entry is booted right away.

Further documentation is available in the gummiboot wiki at:
  http://freedesktop.org/wiki/Software/gummiboot

Links:
  http://www.freedesktop.org/wiki/Specifications/BootLoaderSpec
  http://www.freedesktop.org/software/systemd/man/kernel-install.html
