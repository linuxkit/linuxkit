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
- [Wireguard](wireguard/) cryptographic enforced container network separation
- [OKernel](okernel/) intra-kernel protection using EPT (HPE)
- [eBPF](ebpf/) iovisor eBPF tools
- [AWS](aws/) AWS build support
- [Swarmd](swarmd) Standalone swarmkit based orchestrator
- [Landlock LSM](landlock/) programmatic access control

## Current projects not yet documented
- Clear Linux integration (Intel)
- VMWare support (VMWare)
- ARM port and secure boot integration (ARM)
