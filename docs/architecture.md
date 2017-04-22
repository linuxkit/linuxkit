# Architecture

The `moby` tool converts the yaml specification into one or more bootable images.

A kernel image is a Docker image with a `bzImage` file for the kernel to be executed,
and a tarball `kernel.tar` containing any modules. This is somewhat inconsistent with
the other images (see below) and may be changed. The kernel build is in the
[`kernel/`](../kernel/) directory.

For all other images, ie `init` and the system containers and daemons, the filesystem
of the container is extracted. The `init` and `kernel` filesystems are left unchanged,
while the other containers are currently extracted under the `containers/` directory
where the `init` script runs them. In future they may be extracted to the `containerd`
image store instead. The builds and source for these containers can be found in the
[`pkg/`](../pkg/) directory.

The `init` that is being used is being reworked, as an earlier incarnation was much
less containerised and we ran a full Alpine Linux distribution here. It should end
up as just `init`, a basic setup script, and `containerd`, with `mdev` and `getty`
moved into containers.

The system containers are fairly low level, and most users will probably want to
start with something based on the examples. The aim is to provide a base set that
covers general setup (such as `sysfs` to configure that, and a container to provide
`dhcp` which many platforms use), and a set of platform specific ones, for example
the `metadata-*` ones which will get userdata from cloud providers. There will be
some `dev` containers to allow easy access to the system, for example with `ssh`.
Then there will be some common applications.

For each system or daemon container, an OCI `config.json` file is generated. This is
currently done via a modified version of `riddler` running in a container but will
soon be switched to run directly from the config. This is added to the built image
so it can be called at runtime.

Once all the container filesystems have been unpacked, they are joined into a Linux
initramfs, which is a compressed `cpio` archive. This and the kernel can be booted
directly on some platforms, such as qemu or hyperkit, but other platforms need these
to have a bootloader added, so this is done by the output formats.

The `kernel+initrd` target outputs the raw kernel and initramfs, as well as a file
with the specified command line. It can be used to build other targets or used by
scripts directly.

The output formats are all, except the simple `kernel+initrd` target, generated via
Docker containers, as there are not yet good libraries for outputting these formats
in Go. Most of the current ones create an ISO or ext4 filesystem with `syslinux`
as a boot loader, which directly boots the kernel and initramfs. This is not a requirement
and other bootloaders can be used, and the filesystem could be unpacked onto the
media if required too, or a more complex boot loader scheme used, such as the one
ChromeOS has, with upgrade and fallback facilities.

Because the image is run as an initramfs, and the system containers are
baked in, upgrades are done by updating the system externally. This makes the whole
system immutable, the [phoenix server](https://martinfowler.com/bliki/ImmutableServer.html)
model. Persistent storage can be added using a volume (examples coming soon based on
what the Docker Editions use). For running programs dynamically, a container
orchestrator such as Docker or Kubernetes can be used; simpler distributed applications
can be hard coded into the initramfs if they are suited to being run directly on
immutable infrastructure.

In production we expect most users will use a cluster of instances, usually running
distributed applications, such as `etcd`, `Docker`, `Kubernetes` or distributed
databases. We will provide examples of how to run these effectively, largely using
`infrakit`, although other machine orchestration systems can equally well be used,
for example Terraform or VMWare, and we welcome examples and documentation for those
too.
