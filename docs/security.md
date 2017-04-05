# Security Design

Moby is architected to be secure by default. This document intends to detail the design decisions behind Moby that
pertain to security, as well as provide context for future project direction.


## Modern and Securely Configured Kernels

Moby uses modern kernels, and updates frequently following new releases. It is well understood that many kernel bugs
may lurk in the [codebase for years](https://lwn.net/Articles/410606/). Therefore, it is imperative to not only patch
the kernel to fix individual vulnerabilities but also benefit from the upstream security measures designed to prevent
classes of kernel bugs.

In practice this means Moby tracks new kernel releases very closely, and also follows best practice settings for the
kernel configuration from the [Kernel Self Protection Project](https://kernsec.org/wiki/index.php/Kernel_Self_Protection_Project)
and elsewhere.

The Moby project maintainers are actively collaborating with KSPP and it is an established [priority for the project](../projects/kspp/roadmap.md).

The Moby kernel is intended to be identical to the upstream kernel - We only intend to carry patches that are on track
to be upstreamed, or fix regressions or bugs and that we will upstream.


## Minimal Base

Moby is not a full host operating system, as it primarily has two jobs: run `containerd` containers, and be secure.

As such, the system does not contain extraneous packages or drivers by default. Because Moby is customizable, it is up to
individual operators to include any additional bits they may require.


## Type Safe System Daemons

The core system components that we must include in Moby userspace are key to security, and we believe
they should be written in type safe languages, such as [Rust](https://www.rust-lang.org/en-US/), [Go](https://golang.org/)
and [OCaml](http://www.ocaml.org/), and run with maximum privilege separation and isolation.

The project is currently leveraging [MirageOS](https://mirage.io/) to construct unikernels to achieve this, and that progress can be
[tracked here](../projects/miragesdk/roadmap.md): as of this writing, `dhcp` is the first such type safe program.
There is ongoing work to remove more C components, and to improve, fuzz test and isolate the base daemons.
Further rationale about the decision to rewrite system daemons in MirageOS is explained at length in [this document](../projects/miragesdk/README.md).

For the daemons in which this is not complete, as an intermediate step they are running as `containerd` containers,
and namespaced separately from the host as appropriate.


## Built With Hardened Toolchains and Containers

Moby's build process heavily leverages Docker images for packaging. Of note, all intermediate build images
are referenced by digest to ensures reproducibility across Moby builds. Tags are mutable, and thus subject to override
(intentionally or maliciously) - referencing by digest mitigates classes of registry poisoning attacks in Moby's buildchain.
Certain images, such as the kernel image, will be signed by Moby maintainers using [Docker Content Trust](https://docs.docker.com/engine/security/trust/content_trust/),
which guarantees authenticity, integrity, and freshness of the image.

Moreover, Moby's build process leverages [Alpine Linux's](https://alpinelinux.org/) hardened userspace tools such as
Musl libc, and compiler options that include `-fstack-protector` and position-independent executable output. Go binaries
are also PIE.


## Immutable Infrastructure

Moby runs as an initramfs and its system containers are baked in at build-time, essentially making Moby immutable.

Moreover, Moby has a read-only root filesystem: system configuration and sensitive files cannot be modified after boot.
The only files on Moby that are allowed to be modified pertain to namespaced container data and stateful partitions.

As such, Moby access to the Moby base system is limited in scope: in the event of any container escape, the attack surface
is also limited because the system binaries and configuration is unmodifiable. To that end, the Moby base system does not
supply a package manger: containers must be built beforehand with the dependencies they require.

Once a secure Moby base system has been built, it cannot be tampered with, even by malicious user containers. Even if user
containers unintentionally expose themselves to attack vectors, immutability of the Moby base system limits the scope of
host attack.


## External Updates - Trusted Provisioning

Following the principle of least privilege for immutable infrastructure, Moby cannot have the ability or attack surface 
to update itself. It is the responsibility of an external system, most commonly [infrakit](https://github.com/docker/infrakit), to provision
and update Moby nodes.

It is encouraged to consider the notion of "reverse uptime" when deploying Moby - because Moby is immutable, it should be
acceptable and encouraged to frequently redeploy Moby nodes.

Moby cannot make any trusted hardware assumptions because of the vast variety of platforms it boots on, but Infrakit
can be used to provide trusted boot information and integrate with existing trusted boot hardware. In this sense, Moby is
"trusted boot-ready" and the team is already collaborating with cloud and hardware providers to make this a reality.


## Incubating Next-generation Security Projects

Since Moby is meant to only run containers and be secure, it is the perfect platform to incubate new (and potentially radical!)
paradigms and strategies for securing the Linux kernel - allowing them to be used in production environments and attract
critical mass before eventually being upstreamed.

In this spirit, the [`/projects`](../projects) subdirectory houses a number of such projects. At this time, these include:
- [WireGuard](../projects/wireguard/roadmap.md): a modern and minimal VPN implemented with the state-of-the-art cryptography
like the [Noise protocol framework](http://www.noiseprotocol.org/)
- [okernel](../projects/okernel/roadmap.md): a mechanism to split the kernel into inner and outer subkernels with different trust properties

The Moby community welcomes new security projects - please propose a new project if you have one you'd like to include!
