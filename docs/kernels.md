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
enabled. The hub images for these kernels have the `_dbg` suffix in
the tag. For some kernels, we also provide matching packages
containing the `perf` utility for debugging and performance tracing.
The perf package is called `kernel-perf` and is tagged the same way as
the kernel packages.

In addition to the official kernel images, LinuxKit offers the ability
to build bootable Linux images with kernels from various
distributions. We mostly offer this mostly for testing
purposes. "Foreign" kernel images are created by re-packing the native
kernel packages into hub images. The hub images are typically tagged
with the kernel version.

In summary, LinuxKit offers a choice of the following kernels:
- [linuxkit/kernel](https://hub.docker.com/r/linuxkit/kernel/): Official LinuxKit kernels.
- [linuxkit/kernel-mainline](https://hub.docker.com/r/linuxkit/kernel-mainline/): Mainline [kernel.org](http://kernel.org) kernels from the [Ubuntu Mainline PPA](http://kernel.ubuntu.com/~kernel-ppa/mainline/).
- [linuxkit/kernel-ubuntu](https://hub.docker.com/r/linuxkit/kernel-ubuntu/): Selected Ubuntu kernels.
- [linuxkit/kernel-debian](https://hub.docker.com/r/linuxkit/kernel-debian/): Selected Debian kernels.
- [linuxkit/kernel-centos](https://hub.docker.com/r/linuxkit/kernel-centos/): Selected CentOS kernels.
- [linuxkit/kernel-fedora](https://hub.docker.com/r/linuxkit/kernel-fedora/): Selected Fedora kernels.


## Compiling external kernel modules

This section describes how to build external (out-of-tree) kernel modules. It is assumed you have
the source available to those modules, and require the correct kernel version headers and compile tools.

The LinuxKit kernel packages include `kernel-dev.tar` which contains
the headers and other files required to compile kernel modules against
the specific version of the kernel. Currently, the headers are not
included in the initial RAM disk, but it is possible to compile custom
modules offline and include then include the modules in the initial
RAM disk.

There is a [example](../test/cases/020_kernel/010_kmod_4.9.x), but basically one can use a
multi-stage build to compile the kernel modules:
```
FROM linuxkit/kernel:4.9.33 AS ksrc
# Extract headers and compile module
FROM linuxkit/kernel-compile:1b396c221af673757703258159ddc8539843b02b@sha256:6b32d205bfc6407568324337b707d195d027328dbfec554428ea93e7b0a8299b AS build
COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

# copy module source code and compile
```

To use the kernel module, we recommend adding a final stage to the
Dockerfile above, which copies the kernel module from the `build`
stage and performs a `insmod` as the entry point. You can add this
package to the `onboot` section in your YAML
file. [kmod.yml](../test/cases/020_kernel/010_kmod_4.9.x/kmod.yml) contains an example for the
configuration.

## Compiling internal kernel modules
If you want to compile in-tree kernel modules, i.e. those whose source is already in the
kernel tree but have not been included in `linuxkit/kernel`, you have two options:

1. Follow the external kernel modules process from above
2. Modify the kernel config in [../kernel/](../kernel/) and rebuild the kernel.

In general, if it is an in-tree module, we prefer to include it in the standard linuxkit kernel
distribution, i.e. option 2 above. Once you have it working, please open a Pull Request to include it.

### External Process
The `kernel-dev.tar` included with each kernel does *not* include the kernel sources, *only* the headers.
To build those modules, you will need to download the kernel source separately and recompile. The
in-container process that downloads the source is available in the [Dockerfile](../kernel/Dockerfile).

### Modify Config
Building an in-tree module is very similar to building a new modified kernel (see below):

1. Modify the appropriate `kernel.config-*` file(s)
2. Compile

## Building and using custom kernels

To build and test locally modified kernels, e.g., to try a different
kernel config or new patches, the existing kernel build system in the
[`../kernel`](../kernel/) can be re-used. For example, assuming the
current 4.9 kernel is 4.9.33, you can build a local kernel with:

```
make build_4.9.x
```

This will create a local kernel image called
`linuxkit/kernel:4.9.33-<hash>-dirty` assuming you haven't committed you local changes. You can then use this in your YAML file as:
```
kernel:
  image: linuxkit/kernel:4.9.33-<hash>-dirty
```

If you have committed your local changes, the `-dirty` will not be appended. Then you can also override the Hub organisation to use the image elsewhere with:
```
make ORG=<your hub org>
```
The image will be uploaded to Hub and can be use in a YAML file as
`<your hub org>/kernel:4.9.33` or as `<your hub
org>/kernel:4.9.33-<hash>`.

### Modifying the Config
Each series of kernels has a config file dedicated to it in [../kernel/](../kernel), e.g.
[kernel.config-4.9.x](../kernel/kernel_config-4.9.x). To build a particular series of kernel:

1. Create a separate `git` branch (not required but *strongly* recommended)
2. Modify the appropriate `kernel.config`, e.g. `kernel.config-4.9.x`
3. Run `make build_<series>` with appropriate arguments per this section, e.g. `make build_4.9.x ORG=foo HASH=bar`
4. Create a `.yml`, build and test

You can modify the config in one of two ways:

* Manually, editing the config file
* Using a standard config generator, like `menuconfig`

Generally, you will manually edit a file if you are a Linux kernel expert and _fully_ understand all of the dependencies, or if the change is minor and you are _highly confident_ there are no dependencies.

If you wish to use `menuconfig`, which figures out dependencies for you, you will need an environment in which to run it. Fortunately, the linuxkit project's kernel compile process already sets one up for you.
To get an appropriate environment:

1. `cd kernel/`
2. Run a build for your desired kernel series, e.g. `make build_4.9.x ORG=foo HASH=bar`
3. When you see the output from `make defconfig && make oldconfig` complete, hit `Ctrl-C` to stop the build
4. Note the hash from the intermediate container. That intermediate container has all of the tools and source in it, and can be used to build.
5. Get a shell in that intermediate container, mounting the current directory in: `docker run -it --rm -v ${PWD}:/src <hash> sh`

This will give you a read-to-run kernel build environment, with all of the config files in `/src/`.

For the output of step 4, e.g.:

```
Step X/29 : COMMAND
 ---> b2a4a976d661
```

Once you have your shell, and you want to run the config, you can do the following. We assume you have launched your config container using the steps above, i.e. `docker run -it --rm -v ${PWD}:/src <hash> sh`. The kernel source is in `/linux/`, while the `kernel/` directory from linuxkit is in `/src/`:

Unless you are building the config from scratch, you probably want to make small modifications to the existing config.

The appropriate config at `/src/kernel.config-<series>` was already copied over to `/linux/.config` by the build.

1. `cd /linux`
2. `make menuconfig`
3. Load in the existing config: On the bottom menu, use the left-right arrow keys to `Load`
4. Load it from `.config`
5. `Exit` from the `Load` pop-up and make the desired changes
6. Save the modified config: On the bottom menu, use the left-right arrow keys to `Save`
7. Save it to `.config`
8. Exit the menu by selecting `Exit` from the bottom meny as many times as necessary
9. Copy the saved config to the mount location: `cp /linux/.config /src/some-saved-name.config` (replace with an appropriate name)
10. Exit out of the container
11. Check the differences generated by menuconfig with `diff kernel.config-<series> some-saved-name.config`.
    * If the changes are as you expected, proceed to the next step
    * If the changes are different, either return to the container and menuconfig, or edit manually
12. Copy the new config file to the build location: `cp some-saved-name.config kernel.config-<series>`
13. Run your build: `make build_<series>`



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

Patches are applied to point releases of the linux stable tree. You need an up-to-date copy of that tree:
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

To use the binary, you can either bind mount it into the `getty` or `ssh` service container or you can access the root filesystem from the `getty` container via `nsenter`:
```
nsenter -m/proc/1/ns/mnt ash
```

Alternatively, you can add the `kernel-perf` package as stage in a
multi-stage build to add it to a custom package.
