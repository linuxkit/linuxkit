# Linux kernels

LinuxKit kernel images are distributed as hub images which contain the
kernel, kernel modules, kernel config file, and optionally, kernel
headers to compile kernel modules against. The repository containing
the official LinuxKit kernels is at
[linuxkit/kernels](https://hub.docker.com/r/linuxkit/kernel/).

The LinuxKit kernels are based on the latest stable releases and are
updated frequently to include bug and security fixes.  For some
kernels we do carry additional patches, which are mostly back-ported
fixes from newer kernels. The full kernel source with patches can be
found on [github](https://github.com/linuxkit/linux).

## Kernel Image Naming and Tags

We publish the following kernel images:

* primary kernel
* debug kernel
* tools for the specific kernel build
* builder image for the specific kernel build, useful for compiling compatible kernel modules

### Primary Kernel Images

Each kernel image is tagged with:

* the full kernel version, e.g. `linuxkit/kernel:6.6.13`. This is a multi-arch index, and should be used whenever possible.
* the full kernel version plus hash of the files it was created from (git tree hash of the `./kernel` directory), e.g. `6.6.13-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd`. This is a multi-arch index.
* the full kernel version plus architecture, e.g. `linuxkit/kernel:6.6.13-amd64` or `linuxkit/kernel:6.6.13-arm64`. Each of these is architecture specific.
* the full kernel version plus hash of the files it was created from (git tree hash of the `./kernel` directory) plus architecture, e.g. `6.6.13-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd-arm64`.

### Debug Kernel Images

With each kernel image, we also publish kernels with additional debugging enabled.
These have the same image name and the same tags as the primary kernel, with the `-dbg`
suffix added immediately after the version. E.g.

* `linuxkit/kernel:6.6.13-dbg`
* `linuxkit/kernel:6.6.13-dbg-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd`
* `linuxkit/kernel:6.6.13-dbg-amd64`
* `linuxkit/kernel:6.6.13-dbg-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd-amd64`

### Tools

With each kernel image, we also publish images with various tools. As of this writing,
those tools are `perf` and `bcc`.

The tools images are named `linuxkit/kernel-<tool>`, followed by the same tags as the
primary kernel. For example:

* `linuxkit/kernel-perf:6.6.13`
* `linuxkit/kernel-perf:6.6.13-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd`
* `linuxkit/kernel-perf:6.6.13-amd64`
* `linuxkit/kernel-perf:6.6.13-c0d96951e9892a7447a8e7965d2d6bd7e621c3fd-amd64`

## Additional Contributions

In addition to the official images, there are also some
[scripts](../contrib/foreign-kernels) which repackage kernels packages
from some Linux distributions into LinuxKit kernel packages. These are
mostly provided for testing purposes.

Note now linuxkit also embraces Preempt-RT Linux kernel to support more
use cases for the promising IoT scenarios. All -rt patches are grabbed from
https://www.kernel.org/pub/linux/kernel/projects/rt/. But so far we just
enable it over 4.14.x.

## Loading kernel modules

Most kernel modules are autoloaded with `mdev` but if you need to `modprobe` a module manually you can use the `modprobe` package in the `onboot` section like this:
```
  - name: modprobe
    image: linuxkit/modprobe:<hash>
    command: ["modprobe", "-a", "iscsi_tcp", "dm_multipath"]
```

## Compiling external kernel modules

This section describes how to build external (out-of-tree) kernel
modules. You need the following to build external modules. All of
these are to be built for a specific version of the kernel. For
the examples, we will assume 5.10.104; replace with your desired
version.

* source available to your modules - you need to get those on your own
* kernel development headers - available in the `linuxkit/kernel` image as `kernel-dev.tar`, e.g. `linuxkit/kernel:5.10.104`
* OS with sources and compiler - this **must** be the exact same version as that used to compile the kernel

As described above, the `linuxkit/kernel` images include `kernel-dev.tar` which contains
the headers and other files required to compile kernel modules against
the specific version of the kernel. Currently, the headers are not
included in the initial RAM disk, but it is possible to compile custom
modules offline and then include the modules in the initial RAM disk.

The source is available as the same name as the `linuxkit/kernel` image, with the addition of `-builder` on the tag.
For example:

* `linuxkit/kernel:5.10.92` has builder `linuxkit/kernel:5.10.92-builder`
* `linuxkit/kernel:5.15.15` has builder `linuxkit/kernel:5.15.15-builder`

With the above in hand, you can create a multi-stage `Dockerfile` build to compile your modules.
There is an [example](../test/cases/020_kernel/113_kmod_5.10.x), but
basically one can use a multi-stage build to compile the kernel
modules:

```dockerfile
FROM linuxkit/kernel:5.10.104 AS ksrc
FROM linuxkit/kernel:5.10.104-builder AS build

RUN apk add build-base

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

# copy module source code and compile
```

To use the kernel module, we recommend adding a final stage to the
Dockerfile above, which copies the kernel module from the `build`
stage and performs a `insmod` as the entry point. You can add this
package to the `onboot` section in your YAML
file. [test.yml](../test/cases/020_kernel/113_kmod_5.10.x/test.yml)
contains an example for the configuration.

### Builder Backups

As described above, the OS builder is referenced via `<kernel-image>-builder`, e.g.
`linuxkit/kernel:5.15.15-builder`.

As a fallback, in case the `-builder` image is not available or you cannot access it from your development environment,
you have 3 total places to determine the correct version of the OS image with sources and compiler:

* `-builder` tag added to the kernel version, e.g. `linuxkit/kernel:5.10.104-builder`
* labels on the kernel image, e.g. `docker inspect linuxkit/kernel:5.10.104 | jq -r '.[].Config.Labels["org.mobyproject.linuxkit.kernel.buildimage"]'`
* `/kernel-builder` file in the kernel image

You **should** use `-builder` tag as the `AS build` in your `Dockerfile`, but you **can** use
the direct source, extracted from the labels or `/kernel-builder` file in the kernel image, in the `AS build`.

For example, in the case of `5.10.104`, the label and `/kernel-builder` file show `linuxkit/alpine:2be490394653b7967c250e86fd42cef88de428ba`,
so you can use either `linuxkit/alpine:2be490394653b7967c250e86fd42cef88de428ba` or
`linuxkit/kernel:5.10.104-builder` to build the modules.

Thus, the following are equivalent:

```dockerfile
FROM linuxkit/kernel:5.10.104 AS ksrc
FROM linuxkit/kernel:5.10.104-builder AS build
```

```dockerfile
FROM linuxkit/kernel:5.10.104 AS ksrc
FROM linuxkit/alpine:2be490394653b7967c250e86fd42cef88de428ba AS build
```

## Building and Modifying

This section describes how to build kernels, and how to modify existing ones.

Throughout the document, the terms used are:

* kernel version: actual semver version of a kernel, e.g. `6.6.13` or `5.15.27`
* kernel series: major.minor version of a kernel, e.g. `6.6.x` or `5.15.x`

Each series of kernels has a config file dedicated to it in [../kernel/](../kernel),
e.g. [config-5.10.x-x86_64](../kernel/config-5.10.x-x86_64),
one per target architecture. Note that the architecture used as the `uname -m` one
and not the alpine or golang one. Thus `x86_64` rather than `amd64`, and `aarch64` rather
than `arm64`.

The series+arch config file is applied during the kernel build process.

**Note**: We try to keep the differences between kernel versions and
architectures to a minimum, so if you make changes to one
configuration also try to apply it to the others. The script [kconfig-split.py](../scripts/kconfig-split.py) can be used to compare kernel config files. For example:

```sh
../scripts/kconfig-split.py config-4.9.x-aarch64 config-4.9.x-x86_64
```

creates a file with the common and the x86_64 and arm64 specific
config options for the 4.9.x kernel series.

**Note**: The CI pipeline does *not* push out kernel images.
Anyone modifying a kernel should:

1. Follow the steps below for the desired changes and commit them.
1. Run appropriate `make build` or variants to ensure that it works.
1. Open a PR with the changes. This may fail, as the CI pipeline may not have access to the modified kernels.
1. A maintainer should run `make push` to push out the images.
1. Run (or rerun) the tests.

#### Build options

The targets and variants for building are as follows:

* `make build` - make all kernels in the version list and their variants
* `make build-<version>` - make all variants of a specific kernel version
* `make buildkernel-<version>` - make all variants of a specific kernel version
* `make buildplainkernel-<version>` - make just the provided version's kernel
* `make builddebugkernel-<version>` - make just the provided version's debug kernel
* `make buildtools-<version>` - make just the provided version's tools

To push:

* `make push` - push all kernels in the version list and their variants
* `make push-<version>` - push all variants of a specific kernel version

Finally, for convenience:

* `make list` - list all kernels in the version list

By default, it builds for all supported architectures. To build just for a specific
architecture:

```sh
make build ARCH=amd64
```

The variable `ARCH` should use the golang variants only, i.e. `amd64` and `arm64`.

To build for multiple architectures, call it multiple times:

```sh
make build ARCH=amd64
make build ARCH=arm64
```

When building for a specific architecture, the build process will use your local
Docker, passing it `--platforms` for the architecture. If you have a builder on a different
architecture, e.g. you are running on an Apple Silicon Mac (arm64) and want to build for
`x86_64` without emulating (which can be very slow), you can use the `BUILDER` variable:

```sh
make build ARCH=x86_64 BUILDER=remote-amd64-builder
```

Builder also supports a builder pattern. If `BUILDER` contains the string `{{.Arch}}`,
it will be replaced with the architecture being built.

For example:

```sh
make build ARCH=x86_64 BUILDER=remote-{{.Arch}}-builder
make build ARCH=aarch64 BUILDER=remote-{{.Arch}}-builder
```

will build `x86_64` on `remote-amd64-builder` and `aarch64` on `remote-arm64-builder`.

Finally, if no `BUILDER` is specified, the build will look for a builder named
`linuxkit-linux-{{.Arch}}-builder`, e.g. `linuxkit-linux-amd64-builder` or
`linuxkit-linux-arm64-builder`. If that builder does not exist, it will fall back to
your local Docker setup.

### Modifying the kernel config

The process of modifying the kernel configuration is as follows:

1. Create a `linuxkit/kconfig` container image: `make kconfig`. This is not pushed out.
1. Run a container based on `linuxkit/kconfig`.
1. In the container, modify the config to suit your needs using normal kernel tools like `make defconfig` or `make menuconfig`.
1. Save the config from the image.

The `linuxkit/kconfig` image contains the patched sources
for all support kernels and architectures in `/linux-<major>.<minor>.<rev>`.
The kernel source also has the kernel config copied to the default kernel config location,
so that `make menuconfig` and `make defconfig` work correctly.

Run the container as follows:

```sh
docker run --rm -ti -v $(pwd):/src linuxkit/kconfig
```

This will give you a interactive shell where you can modify the kernel
configuration you want, while mounting the directory, so that you can save the
modified config.

To create or modify the config, you must cd to the correct directory,
e.g.

```sh
cd /linux-6.6.13
# or
cd /linux-5.15.27
```

Now you can build the config.

When `make defconfig` or `make menuconfig` is done,
the modified config file will be in `.config`; save the file back to `/src`,
e.g.

```sh
cp .config /src/kernel-config-6.6.x-x86_64
```

You can also configure other architectures other than the native
one. For example to configure the arm64 kernel on x86_64, use:

```sh
make ARCH=arm64 defconfig
make ARCH=arm64 oldconfig # or menuconfig
```

Note that the generated file **must** be final. When you actually build the kernel,
it will check that running `make defconfig` will have no changes. If there are changes,
the build will fail.

The easiest way to check it is to rerun `make defconfig` inside the kconfig container.

1. Finish your creation of the config file, as above.
1. Copy the `.config` file to the target location, as above.
1. Copy the `.config` file to the source location for defconfig, e.g. `cp .config arch/x86/configs/x86_64_config` or `cp. config /linux/arch/arm64/configs/defconfig`
1. Run `make defconfig` again, and check that there are no changes, e.g. `diff .config arch/x86/configs/x86_64_config` or `diff .config /linux/arch/arm64/configs/defconfig`

If there are no differences, then you can commit the new config file.

Finally, test that you can build the kernel with that config as `make build-<version>`, e.g. `make build-5.15.148`.

## Adding a new kernel version

If you want to add a new kernel version within an existing series, e.g. `5.15.27` already exists
and you want to add (or replace it with) `5.15.148`, apply the following process.

1. Modify the list of kernels inside the `Makefile` to include the new version, and, optionally, remove the old one.
1. Create a new `linuxkit/kconfig` container image: `make kconfig`. This is not pushed out.
1. Run a container based on `linuxkit/kconfig`.
```sh
docker run --rm -ti -v $(pwd):/src linuxkit/kconfig
```
1. In the container, change directory to the kernel source directory for the new version, e.g. `cd /linux-5.15.148`.
1. Run `make defconfig` to create the default config file.
1. If the config file has changed, copy it out of the container and check it in, e.g. `cp .config /src/kernel-config-5.15.x-x86_64`.
1. Repeat for other architectures.
1. Commit the changed config files.
1. Test that you can build the kernel with that config as `make build-<version>`, e.g. `make build-5.15.148`.

## Adding a new kernel series

To add a new kernel series, you need to create a new config file. Since the last major series
likely is the best basis for the new one, subject to additional modifications, you can use
the previous one as a starting point.

1. Modify the list of kernels inside the `Makefile` to include the new version. You do not need to specify the series anywhere, as the `Makefile` calculates it. E.g. adding `7.0.5` will cause it to calculate the series as `7.0.x` automatically.
1. Create a new `linuxkit/kconfig` container image: `make kconfig`. This is not pushed out.
1. Run a container based on `linuxkit/kconfig`.
```sh
docker run --rm -ti -v $(pwd):/src linuxkit/kconfig
```
1. In the container, change directory to the kernel source directory for the new version, e.g. `cd /linux-7.0.5`.
1. Copy the existing config file for the previous series, e.g. `cp /src/kernel-config-6.6.x-x86_64 .config`.
1. Run `make oldconfig` to create the config file for the new series from the old one. Answer any questions.
1. Save the newly generated config file `.config` to the source directory, e.g. `cp .config /src/kernel-config-7.0.x-x86_64`.
1. Repeat for other architectures.
1. Commit the new config files.
1. Test that you can build the kernel with that config as `make build-<version>`, e.g. `make build-7.0.5`.

In addition, there are tests that are applied to a specific kernel version, notably the tests in
[020_kernel](../test/cases/020_kernel/). You will need to add a new test case for the new series,
copying an existing one and modifying it as needed.

## Building and using custom kernels

To build and test locally modified kernels, e.g., to try a different
kernel config or new patches, the existing kernel build system in
the [`kernel`](../kernel/) directory can be re-used. For example,
assuming the current 4.9 kernel is 4.9.33, you can build a local
kernel with:

```sh
make build_4.9.x
```

This will create a local kernel image called
`linuxkit/kernel:4.9.33-<hash>-dirty` assuming you haven't committed
you local changes. You can then use this in your YAML file as:

```
kernel:
  image: linuxkit/kernel:4.9.33-<hash>-dirty
```

If you have committed your local changes, the `-dirty` will not be
appended. Then you can also override the Hub organisation to use the
image elsewhere with (and also disable image signing):

```sh
make ORG=<your hub org>
```

The image will be uploaded to Hub and can be use in a YAML file as
`<your hub org>/kernel:4.9.33` or as `<your hub
org>/kernel:4.9.33-<hash>`.

The kernel build system has some provision to allow local
customisation to the build.

If you want to override/add some kernel config options, you can add a
file called `config-4.9.x-x86_64-foo` and then invoke the build with `make
EXTRA=-foo build_4.9.x-foo` and this will build an image with the
additional kernel config options enabled.

If you want additional patches being applied, just copy them to the
`patches-4.X.x` and the build process will pick them up.


## Working with Linux kernel patches for LinuxKit

We may apply patches to the Linux kernel used in LinuxKit, primarily to
cherry-pick some upstream patches or to add some additional
functionality, not yet accepted upstream.

Patches are located in `kernel/patches-<kernel version>` and should follow these rules:
- Patches *must* be in `git am` format, i.e. they should contain a
  complete and sensible commit message.
- Patches *must* contain a Developer's Certificate of Origin.
- Patch files *must* have a numeric prefix to ensure the ordering in
  which they are applied.
- If patches are cherry-picked, they *must* be cherry-picked with `-x`
  to contain the original commit ID.
- If patches are from a different git tree (other than the stable
  tree), or from a mailing list posting they should contain an
  `Origin:` line with a link to the source.

This document outlines the recommended procedure to handle
patches. The general process is to apply them to a branch of the
[Linux stable tree](https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git/)
and then export them with `git format-patch`.

If you want to add or remove patches currently used, please also ping
@rneugeba on the PR so that we can update our internal Linux tree to
ensure that patches are carried forward if we update the kernel in the
future.


### Preparation

Patches are applied to point releases of the linux stable tree. You
need an up-to-date copy of that tree:

```sh
git clone git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git
```

Add it as a remote to a clone of the [LinuxKit clone](https://github.com/linuxkit/linux).

We use the following variables:
- `KITSRC`: Base directory of LinuxKit repository
- `LINUXSRC`: Base directory of Linux stable kernel repository
e.g.:

```sh
KITSRC=~/src/linuxkit/linuxkit
LINUXSRC=~/src/linuxkit/linux
```

to refer to the location of the LinuxKit and Linux kernel trees.


### Updating the patches to a new kernel version

There are different ways to do this, but we recommend applying the
patches to the current version and then rebase to the new version. We
define the following variables to refer to the current base tag and
the new tag you want to rebase the patches to:

```sh
CURTAG=v4.9.14
NEWTAG=v4.9.15
```

If you don't already have a branch, it's best to import the current
patch set and then rebase:

```sh
cd $LINUXSRC
git checkout -b ${NEWTAG}-linuxkit ${CURTAG}
git am ${KITSRC}/kernel/patches/*.patch
git rebase ${NEWTAG}-linuxkit ${NEWTAG}
```

The `git am` should not have any conflicts and if the rebase has
conflicts resolve them, then `git add <files>` and `git rebase
--continue`.

If you already have linux tree with a `${CURTAG}-linuxkit` branch, you
can rebase by creating a new branch from the current branch and then
rebase:

```sh
cd $LINUXSRC
git checkout ${CURTAG}-linuxkit
git branch ${NEWTAG}-linuxkit ${CURTAG}-linuxkit
git rebase --onto ${NEWTAG} ${NEWTAG} ${NEWTAG}-linuxkit
```

Again, resolve any conflicts as described above.


### Adding/Removing patches

If you want to add or remove patches make sure you have an up-to-date
branch with the currently applied patches (see above). Then either any
normal means (`git cherry-pick -x`, `git am`, or `git commit`, etc) to
add new patches. For cherry-picked patches also please add a `Origin:`
line after the DCO lines with a reference the git tree the patch was
cherry-picked from.

If the patch is not cherry-picked try to include as much information
in the commit message as possible as to where the patch originated
from. The canonical form would be to add a `Origin:` line after the
DCO lines, e.g.:

```
Origin: https://patchwork.ozlabs.org/patch/622404/
```

### Export patches to LinuxKit

To export patches to LinuxKit, you should use `git format-patch` from
the Linux tree, e.g., something along these lines:

```sh
cd $LINUXSRC
rm $KITSRC/kernel/patches-4.9.x/*
git format-patch -o $KITSRC/kernel/patches-4.9.x v4.9.15..HEAD
```

Then, create a PR for LinuxKit.


## Using `perf`

The `kernel-perf` package contains a statically linked `perf` binary
under `/usr/bin` which is matched with the kernel of the same tag.
The simplest way to use the `perf` utility is to add the package to
the `init` section in the YAML file. This adds the binary to the root
filesystem.

To use the binary, you can either bind mount it into the `getty` or
`ssh` service container or you can access the root filesystem from the
`getty` container via `nsenter`:

```sh
nsenter -m/proc/1/ns/mnt ash
```

Alternatively, you can add the `kernel-perf` package as stage in a
multi-stage build to add it to a custom package.


## ZFS

The kernel build Makefile has support for building the ZFS kernel
modules. Note, the modules are currently not distributed as standard
LinuxKit packages and if you wish to use them you have to compile them
yourself:

```sh
cd kernel
make ORG=<foo> push_zfs_4.9.x # or different kernel version
```

will build and push a `zfs-kmod-4.9.<version>` image to Docker Hub
under the `ORG` specified. This package contains the all the standard
kernel modules from the kernel specified plus the `spl` and `zfs`
kernel modules, with `depmod` run over them, so they can be
`modprobe`ed. To use the modules do something like this in your YAML
file:

```
kernel:
  image: linuxkit/kernel:4.9.<version>
  cmdline: "console=tty0 console=ttyS0 console=ttyAMA0"
init:
  - <foo>/zfs-kmod:4.9.<version>
  ...
```

Then, you also need to make sure the Alpine `zfs` utilities are
available in the container where your want to run `zfs` commands. The
Alpine `zfs` utilities are available in `linuxkit/alpine` and the
version of the kernel module should match the version of the
tools. The container where you run the `zfs` tools might also need
`CAP_SYS_MODULE` to be able to load the kernel modules.
