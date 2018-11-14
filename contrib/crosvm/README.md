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
`kernel+squashfs` build format. For example, using `minimal.yml` from
the `./examples` directory, run (but also see the known issues):

```sh
linuxkit build -format kernel+squashfs -decompress-kernel minimal.yml
```

The `-vmlinux` switch is needed since `crosvm` does not grok
compressed linux kernel images.

Then you can run `crosvm`:
```sh
crosvm run --disable-sandbox \
    --root ./minimal-squashfs.img \
    --mem 2048 \
    --socket ./linuxkit-socket \
    minimal-kernel
```

## Known issues

- With 4.14.x, a `BUG_ON()` is hit in `drivers/base/driver.c`. 4.9.x
  kernels seem to work.
- With the latest version, I don't seem to get a interactive console.
- Networking does not yet work, so don't include a `onboot` `dhcpd` service.
- `poweroff` from the command line does not work (crosvm does not seem
  to support ACPI). So to stop a VM you can use the control socket
  and: `./crosvm stop ./linuxkit-socket`
- `crosvm` and its dependencies compile on `arm64` but `crosvm` seems
  to lack support for setting op the IRQ chip on the system I
  tested. I got: `failed to create in-kernel IRQ chip:
  CreateGICFailure(Error(19))`.
