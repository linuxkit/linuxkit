Base repo for Moby, codename for the Docker Linux distro

Initial requirements are being driven by the very minimal goal of replacing boot2docker for the new Mac app.

However these requirements are fairly small and the scope is intended to be much broader.

Simple build instructions: use `make` to build. `make qemu` will boot up in qemu in a container.

You can build for arm, some parts still under development, `make clean` first, then `make qemu-arm` will run in qemu.
