## Docker Hub

There are images on Docker Hub to help with reproducible builds. These are built (by hand) from `alpine/base`,
generally with tags based on the image contents.

- `mobylinux/alpine-base` the base packages for Moby, before we add Docker and our code and config
- `mobylinux/alpine-build-c` for building C code
- `mobylinux/alpine-build-go` for building Go code
- `mobylinux/alpine-bios` for building BIOS image
- `mobylinux/alpine-efi` for building efi images
- `mobylinux/alpine-qemu` for Qemu eg for the tests
- `mobylinux/debian-build-kernel` for the kernel builds while we cannot use Alpine

The `Dockerfile`s for these are under `alpine/base`. To update, modify the `Dockerfile` if you wish
to change the packages used, and `repositories` if needed, and run `make`. This will push the image
to Docker Hub if it has changed, and update `packages` with a list of the current versions. If the
image has changed, update the other Dockerfiles with this base image to use this new tag.

For example, `alpine/base/alpine-base` is the image used to build the Moby image itself, which is
used in `alpine/Dockerfile`. The uploaded tags can be seen at [Docker Hub](https://hub.docker.com/r/mobylinux/alpine-base/tags/).

In addition
- `mobylinux/media` contains build artifacts

You can upload build artifacts with `make media` or `make media MEDIA_PREFIX=1.12.0-` if you want to change the prefix of the git hash.
The will by default be prefixed by `experimental-` if they are Docker experimental builds. These are used in the Mac and Windows build
process to get the images.

Ping @justincormack if you need access to the Hub organization.
