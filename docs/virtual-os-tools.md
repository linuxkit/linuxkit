# Virtualised OS Tools
A number of [hypervisors](https://en.wikipedia.org/wiki/Hypervisor) provide tools or agents that can be installed inside of a virtualised operating system. These tools can consist of system daemons, kernel modules and general utilities that will provide a better experience (either through improvments of **performance**, **management** or **user interaction**) for the virtualised operating system.

## Typical Features
*As every tool offers different feature sets, this summary details the more important features for OS tools.*

* Capability to issue virtual machine power operations from within the Operating System, allowing application shutdown scripts to be ran.
* Synchronisation of clocks between the hypervisor and the virtualised Operating System.
* Capturing OS specific details such as a dynamically allocated IP address and exposing that to the virtualised management system or API.
* Freezing of virtualised Operating System for the hypervisor to take VM snapshots.
* Heart beat functionality from OS to the hypervisor, enabling the hypervisor to determine if the OS within the virtual machine has crashed.
* Execution of commands or scripts within the Operating System.

## Security Concerns
As with all agents and additional tooling installed within an Operating System there is the concern of new attack vectors. With this category of tooling the majority of security issues are mitigated by the design that only the hypervisor can call upon these features. This essentially means that in order to exploit any of this exposed functionality one must first have access to or exploit the hypervisor, at which point the concerns around the tooling are moot.

Also the use of these OS tools is **optional**, meaning that they can be enabled on a case-by-case basis. 

## Tools
### VMware tools / open-vm-tools
Website(s): [VMware Tools](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) / [open-vm-tools](https://github.com/vmware/open-vm-tools)

Usage:

```
services:
  - name: open-vm-tools
    image: linuxkit/open-vm-tools:<HASH>
```

### Qemu GuestAgent
Website: [GuestAgent](http://wiki.qemu.org/Features/GuestAgent)
Usage:

```
services:
[TBD]
```

### Linux Integration Services for Hyper-V
Website : [hvtools](https://www.microsoft.com/en-us/download/details.aspx?id=51612)
Usage:

```
services:
[TBD]
