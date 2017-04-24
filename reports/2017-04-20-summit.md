# DockerCon Internals Summit

## LinuxKit Summit BoF: Session 1

### Incoming changes:
  - Moby tool will move out of LinuxKit
  - init will get replaced and probably re-written in Rust
  - package and kernel builds will leverage multi-stage builds

### Discussion:
- How to combine kernel flavors?  ex: Wireguard + Landlock
  - proposal: Automated kernel patch merging for build recipes?
  - proposal: "Security" kernel channel with combination of projects
- Can we have a LinuxKit mailing list or use Slack?
  - prefer mailing list
  - community `#linuxkit` channel currently exists
- Issue tracking: make the development process more transparent
  - need to keep community up to date with timelines for big changes such as Moby tool movement, can use mailing list
  - discussed creating an issue tracker/milestone with a monthly horizon for tracking progress
- Modules: current kernel won’t change much
- Lessons learned from RancherOS: need robust testing across *all* platforms and use-cases
  - with LinuxKit: instead, have to build custom OS yourself
- Roadmap: replace system daemons with type-safe languages, potentially in unikernels
  - Collaborate with formal-verification groups at Microsoft
- Moby yml: should `outputs` be in the cmd-line vs. yml?
- Diagnostics: currently closed source but will become a public package
- Error logging: we've added yml validation, would love more input from the community with how we can improve this
- Outputs: currently support `bzImage+initrd`, `iso-bios`, `iso-uefi`, `qemu`, `gcp` tarball+upload, `vmdk`, `vhd`, `pxe`
  - Coming soon: `ami`, `azure`
  - Should Moby tool be able to upload outputs to clouds?
    - yes: we might call that `moby push` or something
  - TBD: can we collaborate here with packer?
- Who are the users?
  - Docker: we want to build a lean and secure Linux layer for Docker to run on
  - Custom kernel and appliance builders: using bleeding edge security features, minimal OSes for embedded use-cases, etc.
  - Testers: can easily build and spin up custom Linux OS for reproducible and minimal test environments
  - Tinkerers: folks who want to play, implement weekend projects
- Security: prove value of incubation projects and kernel patches - ultimately want to see our projects make it upstream

