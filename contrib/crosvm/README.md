The Chrome OS Virtual Machine Monitor
[`crosvm`](https://chromium.googlesource.com/chromiumos/platform/crosvm/)
is a lightweight VMM written in Rust. It runs on top of KVM and
optionally runs the device models in separate processes isolated with
seccomp profiles.


## Build/Install

The `Makefile` and `Dockerfile` compile `crosvm` and a suitable
version of `libminijail`. To build:

```sh
make
```

You should end up with a `crosvm` and `libminijail.so` binaries as
well as the seccomp profiles in `./build`. Copy `libminijail.so` to
`/usr/lib` or wherever `ldd` picks it up. You may also need `libcap`
(on Ubuntu or Debian `apt-get install -y libcap-dev`).

You may also have to create an empty directory `/var/empty`.


## Use with LinuxKit images

You can build a LinuxKit image suitable for `crosvm` with the
`kernel+squashfs` build format. For example, using this LinuxKit
YAML file (`minimal.yml`):

```
kernel:
  image: linuxkit/kernel:4.9.91
  cmdline: "console=tty0 console=ttyS0 console=ttyAMA0"
init:
  - linuxkit/init:v0.3
  - linuxkit/runc:v0.3
  - linuxkit/containerd:v0.3
services:
  - name: getty
    image: linuxkit/getty:v0.3
    env:
      - INSECURE=true
trust:
  org:
    - linuxkit
```

run:

```sh
linuxkit build -output kernel+squashfs minimal.yml
```

The kernel this produces (`minimal-kernel`) needs to be converted as
`crosvm` does not grok `bzImage`s. You can convert the LinuxKit kernel
image with
[extract-vmlinux](https://raw.githubusercontent.com/torvalds/linux/master/scripts/extract-vmlinux):

```sh
extract-vmlinux minimal-kernel > minimal-vmlinux
```

Then you can run `crosvm`:
```sh
./crosvm run --seccomp-policy-dir=./seccomp/x86_64 \
    --root ./minimal-squashfs.img \
    --mem 2048 \
    --multiprocess \
    --socket ./linuxkit-socket \
    minimal-vmlinux
```

## Known issues

- With 4.14.x, a `BUG_ON()` is hit in `drivers/base/driver.c`. 4.9.x
  kernels seem to work.
- Networking does not yet work, so don't include a `onboot` `dhcpd` service.
- `poweroff` from the command line does not work (crosvm does not seem
  to support ACPI). So to stop a VM you can use the control socket
  and: `./crosvm stop ./linuxkit-socket`
- `crosvm` and its dependencies compile on `arm64` but `crosvm` seems
  to lack support for setting op the IRQ chip on the system I
  tested. I got: `failed to create in-kernel IRQ chip:
  CreateGICFailure(Error(19))`.
