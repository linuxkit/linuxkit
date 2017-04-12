# LinuxKit

LinuxKit, a toolkit for building custom minimal, immutable Linux distributions.

- Secure defaults without compromising usability
- Everything is replaceable and customisable
- Immutable infrastructure applied to building Linux distributions
- Completely stateless, but persistent storage can be attached
- Easy tooling, with easy iteration
- Built with containers, for running containers
- Designed for building and running clustered applications, including but not limited to container orchestration such as Docker or Kubernetes
- Designed from the experience of building Docker Editions, but redesigned as a general purpose toolkit
- Designed to be managed by external tooling, such as [Infrakit](https://github.com/docker/infrakit) or similar tools
- Includes a set of longer term collaborative projects in various stages of development to innovate on kernel and userspace changes, particularly around security

## Getting Started

### Build the `moby` tool

Simple build instructions: use `make` to build. This will build the customisation tool in `bin/`. Add this
to your `PATH` or copy it to somewhere in your `PATH` eg `sudo cp bin/moby /usr/local/bin/`.

If you already have `go` installed you can use `go get -u github.com/docker/moby/src/cmd/moby` to install
the `moby` tool, and then use `moby build linuxkit.yml` to build the example configuration. You
can use `go get -u github.com/docker/moby/src/cmd/infrakit-instance-hyperkit` to get the
hyperkit infrakit tool.

Once you have built the tool, use `moby build linuxkit.yml` to build the example configuration,
and `bin/moby run linuxkit` to run locally. Use `halt` to terminate on the console.

Build requirements:
- GNU `make`
- GNU or BSD `tar` (not `busybox` `tar`)
- Docker

### Booting and Testing

You can use `moby run <name>` to execute the image you created with `moby build <name>.yml`.
This will use a suitable backend for your platform or you can choose one, for example VMWare.
See `moby run --help`.

Some platforms do not yet have `moby run` support, so you can use `./scripts/qemu.sh moby-initrd.img moby-bzImage moby-cmdline`
or `./scripts/qemu.sh mobylinux-bios.iso` which runs qemu in a Docker container.

`make test` or `make test-hyperkit` will run the test suite

There are also docs for booting on [Google Cloud](docs/gcp.md); `./bin/moby run --gcp <name>.yml` should
work if you specified a GCP image to be built in the config.

More detailed docs will be available shortly, for running both single hosts and clusters.

## Building your own customised image

To customise, copy or modify the [`linuxkit.yml`](linuxkit.yml) to your own `file.yml` or use one of the [examples](examples/) and then run `moby build file.yml` to
generate its specified output. You can run the output with `moby run file`.

The yaml file specifies a kernel and base init system, a set of containers that are built into the generated image and started at boot time. It also specifies what
formats to output, such as bootable ISOs and images for various platforms.

### Yaml Specification

The yaml format specifies the image to be built:

- `kernel` specifies a kernel Docker image, containing a kernel and a filesystem tarball, eg containing modules. The example kernels are built from `kernel/`
- `init` is the base `init` process Docker image, which is unpacked as the base system, containing `init`, `containerd`, `runc` and a few tools. Built from `pkg/init/`
- `onboot` are the system containers, executed sequentially in order. They should terminate quickly when done.
- `services` is the system services, which normally run for the whole time the system is up
- `files` are additional files to add to the image
- `outputs` are descriptions of what to build, such as ISOs.

For a more detailed overview of the options see [yaml documentation](docs/yaml.md).

## Architecture and security

There is an [overview of the architecture](docs/architecture.md) covering how the system works.

There is an [overview of the security considerations and direction](docs/security.md) covering the security design of the system.

## Roadmap

This project was extensively reworked from the code we are shipping in Docker Editions, and the result is not yet production quality. The plan is to return to production
quality during Q2 2017, and rebase the Docker Editions on this open source project.

This is an open project without fixed judgements, open to the community to set the direction. The guiding principles are:
- Security informs design
- Infrastructure as code: immutable, manageable with code
- Sensible secure and well tested defaults
- An open, pluggable platform for diverse use cases
- Easy to use and participate in the project
- Built with containers, for portability and reproducibility
- Run with system containers, for isolation and extensibility
- A base for robust products

## Development reports

There are weekly [development reports](reports/) summarizing work carried out in the week.

## FAQ

See [FAQ](docs/faq.md).
