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
found on [github](https://github.com/linuxkit/linux). Each kernel
image is tagged with the full kernel version (e.g.,
`linuxkit/kernel:4.9.33`) and with the full kernel version plus the
hash of the files it was created from (git tree hash of the `./kernel`
directory). For selected kernels (mostly the LTS kernels and latest
stable kernels) we also compile/push kernels with additional debugging
enabled. The hub images for these kernels have the `-dbg` suffix in
the tag. For some kernels, we also provide matching packages
containing the `perf` utility for debugging and performance tracing.
The perf package is called `kernel-perf` and is tagged the same way as
the kernel packages.

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
modules. It is assumed you have the source available to those modules,
and require the correct kernel version headers and compile tools.

The LinuxKit kernel packages include `kernel-dev.tar` which contains
the headers and other files required to compile kernel modules against
the specific version of the kernel. Currently, the headers are not
included in the initial RAM disk, but it is possible to compile custom
modules offline and then include the modules in the initial RAM disk.

There is a [example](../test/cases/020_kernel/010_kmod_4.9.x), but
basically one can use a multi-stage build to compile the kernel
modules:

```
FROM linuxkit/kernel:4.9.33 AS ksrc
FROM linuxkit/alpine:<hash> AS build
RUN apk add build-base

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

# copy module source code and compile
```

To use the kernel module, we recommend adding a final stage to the
Dockerfile above, which copies the kernel module from the `build`
stage and performs a `insmod` as the entry point. You can add this
package to the `onboot` section in your YAML
file. [kmod.yml](../test/cases/020_kernel/010_kmod_4.9.x/kmod.yml)
contains an example for the configuration.


## Modifying the kernel config

Each series of kernels has a config file dedicated to it
in [../kernel/](../kernel),
e.g.
[config-4.9.x-x86_64](../kernel/config-4.9.x-x86_64),
which is applied during the kernel build process.

If you need to modify the kernel config, `make kconfig` in
the [kernel](../kernel) directory will create a local
`linuxkit/kconfig` Docker image, which contains the patched sources
for all support kernels and architectures in
`/linux-4.<minor>.<rev>`. The kernel source also has the kernel config
copied to the default kernel config.

Running the image like:

```sh
docker run --rm -ti -v $(pwd):/src linuxkit/kconfig
```

will give you a interactive shell where you can modify the kernel
configuration you want, either by editing the config file, or via
`make menuconfig` etc. Once you are done, save the file as `.config`
and copy it back to the source tree,
e.g. `/src/kernel-config-4.9.x-x86_64`.

You can also configure other architectures other than the native
one. For example to configure the arm64 kernel on x86_64, use:

```
make ARCH=arm64 defconfig
make ARCH=arm64 oldconfig # or menuconfig
```

**Note**: We try to keep the differences between kernel versions and
architectures to a minimum, so if you make changes to one
configuration also try to apply it to the others. The script [kconfig-split.py](../scripts/kconfig-split.py) can be used to compare kernel config files. For example:

```sh
../scripts/kconfig-split.py config-4.9.x-aarch64 config-4.9.x-x86_64
```

creates a file with the common and the x86_64 and arm64 specific
config options for the 4.9.x kernel series.

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
make ORG=<your hub org> NOTRUST=1
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
make ORG=<foo> NOTRUST=1 push_zfs_4.9.x # or different kernel version
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
