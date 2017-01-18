Moby, the Linux distro for Docker editions

Simple build instructions: use `make` to build. `make qemu` will boot up in qemu in a container.

Requires GNU `make`, GNU `tar` (not Busybox tar), Docker to build.

- 1.12.x branch for Desktop stable 1.12 edition
- 1.13.x branch for Desktop and Cloud 1.13; also supports 1.12 CS.
- master for 1.14 development

Several kernel variants are supported:
- default
- `make LTS4.4=1` 4.4 LTS series
- `make AUFS=1` supports AUFS (deprecated)
