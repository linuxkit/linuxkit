### Mirage SDK

The goal of the MirageSDK project is to create a set of 'Moby native' system
containers that are secure-by-default and specialised for the Moby execution
environment.  These system containers will implement core system services such
as DHCP, NTP or DNS, with the following properties:

- run in a container as a single static binary.
- follow a common configuration convention based on bind mounts from the host.
- obey strict security conventions:
 * the container has the minimal capabilities required to execute.
 * after configuration is read, the service privilege separates itself to drop as much as possible.
 * processes use KVM to supply extra hardware protection if available, via the Solo5 unikernel.
 * if KVM is not available, use seccomp-bpf to restrict the set of syscalls used.
 * all untrusted network traffic must be handled in memory-safe languages.
 * support automated fuzz testing so that tools like AFL can run regularly to detect bugs proactively.

The SDK will initially support OCaml (via MirageOS), and later expand to cover
Rust. Depending on community interest, we may expand the set of supported
languages, possibly beginning with server-side WebAssembly
([WASM](http://webassembly.org)).  We will not directly support C or other
memory-unsafe languages except through a WASM-style sandbox or interpreter.

Protocol daemons built using the SDK are not intended to be portable to other
Linux or OS distributions, and instead specialised to whatever security
features we include in Moby.  However, we will support and encourage portable
patchsets to be maintained by the community for other operating systems and
distributions, in a similar style to how
[OpenSSH](https://www.openssh.com/portable.html) is maintained.

Why do this?  So far, Moby Linux has packaged up existing system daemons and
moved them into containers. However, many of these daemons continue to suffer
from [security issues](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-1503) whose
root causes are well-understood but not [addressed](https://en.wikipedia.org/wiki/Memory_safety).
MirageSDK aims to be proactively secure-by-default and not burdened with
OS portability code that complicates system interfaces, and to be actively
compatible with the latest security scanning techniques such as fuzz testing,
and to use the best-of-breed privilege separation (such as KVM unikernels) if
the hardware support is available.

# Status

- The first daemon being developed is a DHCP client. This is a difficult daemon to
  privilege separate due the deep (and non-portable) system hooks required for handling
  IP and routing tables (e.g. via `RT_NETLINK`).  Thus this implementation flushes out
  a lot of architectural questions and makes subsequent protocol implementations such
  as HTTPS or NTP more straightforward.  See [why-dhcp](why-dhcp.md) for more details.

- The **[roadmap](roadmap.md)** describes the architecture of the DHCP client and current
  development directions.

- We are also packaging up the Alpine `dhcpcd` with the same configuration conventions
  as the MirageSDK replacement, so that they can swapped in a `linuxkit build` with a single
  line change in the YAML file.

- We will engage external reviewers on the security architecture once we have the first
  clients passing a few hours of AFL fuzz testing and booting on several clouds.

- Documentation and API for further use of the SDK will be published as soon as we have
  a stable DHCP client.  In the meanwhile, please contact us on Docker Community Slack
  in `#moby` or via an issue in this repository with any questions.

# Getting Started

```
../../bin/linuxkit build examples/mirage-dhcp.yml`
../../bin/linuxkit run mirage-dhcp
```
