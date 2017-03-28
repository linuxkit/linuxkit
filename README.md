# Moby

Moby, a toolkit for building custom minimal, immutable Linux distributions.

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

Simple build instructions: use `make` to build.
This will build the Moby customisation tool and an example Moby initrd image from the `moby.yml` file.

If you already have a Go build environment and installed the source in your `GOPATH`
you can do `go install github.com/docker/moby/src/cmd/moby` to install the `moby` tool
instead, and then use `moby build moby.yml` to build the example configuration.

Build requirements:
- GNU `make`
- GNU or BSD `tar` (not `busybox` `tar`)
- Docker

### Booting and Testing

If you have a recent version of Docker for Mac installed you can use `moby run <name>` to execute the image you created with `moby build <name>.yml`

The Makefile also specifies a number of targets:
- `make qemu` will boot up a sample Moby in qemu in a container
- on OSX: `make hyperkit` will boot up Moby in hyperkit
- `make test` or `make hyperkit-test` will run the test suite
- There are also docs for booting on [Google Cloud](docs/gcp.md)
- More detailed docs will be available shortly, for running single hosts and clusters.

## Building your own customised image

To customise, copy or modify the [`moby.yml`](moby.yml) to your own `file.yml` or use one of the [examples](examples/) and then run `./bin/moby build file.yml` to
generate its specified output. You can run the output with `./scripts/qemu.sh` or on OSX with `./bin/moby run file`. `moby run` targets will be available for other
platforms shortly.

The yaml file specifies a kernel and base init system, a set of containers that are built into the generated image and started at boot time. It also specifies what
formats to output, such as bootable ISOs and images for various platforms.

### Yaml Specification

The yaml format is loosely based on Docker Compose:

- `kernel` specifies a kernel Docker image, containing a kernel and a filesystem tarball, eg containing modules. `mobylinux/kernel` is built from `kernel/`
- `init` is the base `init` process Docker image, which is unpacked as the base system, containing `init`, `containerd`, `runc` and a few tools. Built from `base/init/`
- `system` are the system containers, executed sequentially in order. They should terminate quickly when done.
- `daemon` is the system daemons, which normally run for the whole time
- `files` are additional files to add to the image
- `outputs` are descriptions of what to build, such as ISOs.

For the images, you can specify the configuration much like Compose, with some changes, eg `capabilities` must be specified in full, rather than `add` and `drop`, and
there are no volumes only `binds`.

The config is liable to be changed, and there are missing features; full documentation will be available shortly.


## Architecture

There is an [overview of the architecture](architecture/) covering how the system works.

## Roadmap

This project was extensively reworked from the code we are shipping in Docker Editions, and the result is not yet production quality. The plan is to return to production
quality during Q2 2017, and rebase the Docker Editions on this open source project.

Security by default is a key aim. In the short term this means Moby uses modern kernels, best practise settings for the kernel from [KSPP](https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project)
and elsewhere, and a minimal and immutable base. It also means working to incorporate more security features into the kernel, including those in our [projects](projects/). In userspace, the core system components
are key to security, and we believe they should be written in type safe languages, such as Rust, Go and OCaml, and run with maximum privilege separation and isolation.
There is ongoing work to remove C components, and to improve, fuzz test and isolate the base daemons.

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
