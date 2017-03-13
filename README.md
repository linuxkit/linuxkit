# Moby

Moby, a toolkit for custom Linux distributions

## Getting Started

### Build

Simple build instructions: use `make` to build.
This will build the Moby customisation tool and a Moby initrd image.

#### Requirements:

- GNU `make`
- GNU or BSD `tar` (not Busybox tar)
- Docker

### Booting and Testing

- `make qemu` will boot up a sample Moby in qemu in a container
- on OSX: `make hyperkit` will boot up Moby in hyperkit, and also download hyperkit and vpnkit binaries for later use
- `make test` or `make hyperkit-test` will run the test suite

## Customise

To customise, copy or modify the [`moby.yaml`](moby.yaml) to your own `file.yaml` and then run `./bin/moby file.yaml` to
generate its specified output. You can run the output with `./scripts/qemu.sh` or `./scripts/hyperkit.sh`.

### Yaml Specification

The Yaml format is loosely based on Docker Compose:

- `kernel` specifies a kernel Docker image, containing a kernel and a filesystem tarball, eg containing modules. `mobylinux/kernel` is built from `kernel/`
- `init` is the base `init` process Docker image, which is unpacked as the base system, containing `init`, `containerd`, `runc` and a few tools. Built from `base/init/`
- `system` are the system containers, executed sequentially in order. They should terminate quickly when done.
- `daemon` is the system daemons, which normally run for the whole time
- `files` are additional files to add to the image
- `outputs` are descriptions of what to build, such as ISOs.

For the images, you can specify the configuration much like Compose, with some changes, eg `capabilities` must be specified in full, rather than `add` and `drop`, and
there are no volumes only `binds`.

The config is liable to be changed, eg there are missing features (specification of kernel command line, more options etc).
