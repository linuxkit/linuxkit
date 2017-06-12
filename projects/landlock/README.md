# Landlock LSM

[Landlock](https://lwn.net/Articles/698226/) is a Linux Security Module currently under development by Mickaël Salaün (@l0kod). Landlock is based on eBPF,
extended Berkeley Packet filters (see [ebpf project](../ebpf/roadmap.md)), to attach small programs to hooks in the kernel.

These eBPF programs provide context that can allow for very robust decision-making when integrated with LSM hooks.
In particular, this lends itself very nicely to container-based environments.
One such example is that Landlock could be used to write policies to restrict containers from accessing file descriptors they do
not own, acting as a last line of defense to restrict container escapes, 

Landlock is stackable on top of other LSMs, like SELinux and Apparmor.


## Roadmap

**Near-term:**
- We will carry the Landlock patches in a `kernel-landlock` image for people to test, and update them continuously
- Draft and include a simple Landlock policy that can be demonstrated with the current patch-set, to show an example
- Offer design and code review help on Landlock, using Moby as a test-bed

**Long-term:**
- Develop a robust container-minded Landlock policy, and include it in LinuxKit by default
