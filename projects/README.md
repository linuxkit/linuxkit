# Projects

We aim to provide a set of open spaces for collaboration to help move projects towards production. Projects should usually
at a minimum provide a `README` of how to get started using the project with Moby, and a roadmap document explaining what
the aims are and how to contribute. Most projects will probably provide a way to run the project in a custom Moby build
in its current state, which ideally will be integrated in the Moby CI so there are checks that it builds and runs. Over
time we hope that many projects will graduate into the recommended production defaults, but other projects may remain as
ongoing projects, such as kernel hardening.

If you want to create a project, please submit a pull request to create a new directory here.

## Current projects
- [Kernel Self Protection Project enhancements](kspp/)
- [Mirage SDK](miragesdk/) privilege separation for userspace services
- [OKernel](okernel/) intra-kernel protection using EPT (HPE)
- [eBPF](ebpf/) iovisor eBPF tools
- [Landlock LSM](landlock/) programmatic access control
- [Clear Containers](clear-containers/) Clear Containers image
- [Logging](logging/) Experimental logging tools
- [IMA-namespace](ima-namespace/) patches for supporting per-mount-namespace
  IMA policies
- [shiftfs](shiftfs/) is a filesystem for mapping mountpoints across user
  namespaces
- [Memorizer](memorizer/) is a tool to trace intra-kernel
  memory operations.

## Current projects not yet documented
- VMWare support (VMWare)
- ARM port and secure boot integration (ARM)

## Completed projects

- `aws/`: AWS support was merged into mainline in #1964.
- `wireguard/`: [WireGuard](https://www.wireguard.com/) is now part of the default LinuxKit kernel and package set.
- `kubernetes/`: Has been moved to https://github.com/linuxkit/kubernetes.
