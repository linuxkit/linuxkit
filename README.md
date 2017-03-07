Moby, a toolkit for custom Linux distributions

Simple build instructions: use `make` to build.

`make qemu` will boot up a sample in qemu in a container; on OSX `make hyperkit` will
boot up in hyperkit. `make test` or `make hyperkit-test` will run the test suite.

Requires GNU `make`, GNU or BSD `tar` (not Busybox tar) and Docker to build.

To customise, copy or modify the `moby.yaml` and then run `./bin/moby file.yaml` to
generate. You can run the output with `./scripts/qemu.sh` or `./scripts/hyperkit.sh`.

The Yaml format is loosely based on Docker Compose:

- `kernel` specifies a kernel Docker image, containing a kernel and a filesystem tarball, eg containing modules. `mobylinux/kernel` is built from `kernel/`
- `init` is the base `init` process Docker image, which is unpacked as the base system, containing `init`, `containerd`, `runc` and a few tools. Built from `base/init/`
- `system` are the system containers, executed sequentially in order. They should terminate quickly when done.
- `daemon` is the system daemons, which normally run for the whole time
- `files` are additional files to add to the image
- `outputs` are descriptions of what to build, such as ISOs.

For the images, you can specify the configuration much like Compose, with some changes, eg `capabilities` must be specified in full, rather than `add` and `drop`, and
there are no voluems only `binds`.

The config is liable to be changed, eg there are missing features (specification of kernel command line, more options etc).
