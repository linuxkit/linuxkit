Authors: Chris Dalton <cid@hpi.com>, Nigel Edwards <nigel.edwards@hpe.com>,
Theo Koulouris <theo.koulouris@hpe.com>

# Split Kernel

Project links:
- okernel sources on GitHub: https://github.com/linux-okernel/linux-okernel
- Userspace components and supporting material:
  https://github.com/linux-okernel/linux-okernel-components

Similar to the nested-kernel work for BSD by Dautenhahn et al[1], the aim
of the split kernel (okernel) is to introduce a level of intra-kernel
protection into the kernel so that, amongst other things, we can offer
lifetime guarantees over kernel code and data integrity.  Unlike the BSD-
based nested kernel work, we are focused on the Linux kernel (not BSD) and
do make use of HW virtualization features such as Extended Page Tables
(EPT) or equivalent to provide protection from malicious kernel
changes. (Our initial prototype is based on Intel x86, but the
intention is to be architecture neutral so we can apply it to other
architectures, including AMD and ARM.)

The split kernel provides a (protected) virtualized view of the kernel
for processes entering the kernel through exceptions, syscalls and
interrupts. Though we make use of hardware features designed to
support virtualization, we do not virtualize at the full virtual
machine level (like KVM or VMware, for example).  Instead conceptually
our model is closer to the approach prototyped by the DUNE[2] project
where they virtualize much higher up at the user space process
level. DUNE uses the hardware virtualization features to support
virtualization within the user space context of a Linux process to
safely expose privileged hardware features to user programs. We
instead take a cut-line lower down in the OS stack and include the
virtualization of the kernel space context of a process.  This kernel
virtualization allows us to introduce a level of intra-kernel
protection into the Linux kernel.

Our initial prototype consists of a combination of fairly extensive
modifications to the existing DUNE Linux kernel module (which itself
derives from KVM) and a relatively small number of select
modifications to the core Linux kernel code to support the virtualized
kernel cut-line.

In terms of operation, a process can be switched into 'outer-kernel'
mode which includes creating an EPT 'container' (lower level set of
page tables) for it. After switching, the process resumes running in a
non-root (NR) mode VMCS context even when in kernel context.

(In the remainder of this README we use root-mode or R-mode to
describe a process which is has full visibility of the page tables:
upper and lower. NR-mode or non-root mode describes a process which
only has visibility of the upper level page tables.)

With this model, the majority of kernel code can be run within the EPT
'container', offering an enhanced memory protection mechanism whilst
maintaining a single shared kernel image. A small handler loop within
the kernel for each process (thread) handles transitions from NR-mode
to R-mode where necessary to support VMEXITS and provide a privileged
operations interface.

Once a process is in NR-mode, the ability to make changes to kernel
memory is controlled by permissions on both the upper and lower level
page tables. Our security goal is to use the lower level page tables
to prevent a NR-mode process making malicious changes to the
kernel. For example, as far as possible it should not be able to write
code or data pages NR-mode, or if changes are made, they are isolated
to the NR-mode context.

If a process in NR-mode attempts to change the kernel memory in
conflict with permissions in the lower-level page tables, a VMEXIT (in
the current prototype which uses Intel VMX) is triggered. R-mode is
then entered where the permission violation can be handled.

# Integration with LinuxKit

Custom Linux distributions utilizing the split kernel can be readily built
using LinuxKit by simply specifying an okernel image in the `kernel` section
of the LinuxKit OS image YAML specification. See the sample YAML files provided
in [examples/](https://github.com/linuxkit/linuxkit/tree/master/projects/okernel/examples) for more details.

## Building the split kernel image for LinuxKit

 - `make` will build and package the latest version of the split kernel
   for LinuxKit, by    pulling sources from the top-of-tree of the
   okernel project GitHub    (https://github.com/linux-okernel/linux-okernel).
 - Alternatively, a specific version of the kernel can be built
   by setting the 'KERNEL' environment variable to the appropriate
   value, e.g.: `make KERNEL=ok-4.13-rc7`. The value MUST correspond
   to a legitimate okernel tag present in the project GitHub
   (https://github.com/linux-okernel/linux-okernel/tags) beginning
   with __"ok-"__.
   `make KERNEL=latest` will build the top-of-tree release, equivalent to `make`.

`make kvmod` or `make KERNEL=NNNNNNNN kvmod` where "NNNNNNNN" is the release
string corresponding to a kernel version, will build the kernel
vulnerability emulation kernel module for that kernel, useful for testing.

## Enabling NR-mode protections in LinuxKit
By default, okernel will treat processes like any standard Linux kernel, i.e. no
R-mode/NR-mode split is applied when processes are created. This is by design to
assist development and enable coexistence of normal and protected processes in
the same environment. A consequence of that is that processes must be explicitly
launched in NR-mode.

In early versions of LinuxKit we provided a lightly-modified version of the
default 'init' container to ensure runc launched in NR-mode and by extension all
its children processes/containers got instantiated in NR-mode as well.

Recent changes to how LinuxKit implements init made this approach obsolete. We
are in the process of developing an alternative (and hopefully more elegant)
way to launch containers in NR-mode (Aug 2017).

Note that because okernel utilizes the processor's VMX facilities, okernel
LinuxKit images with full NR-mode operation can only be run in baremetal
environments. If no processes are switched to NR-mode, okernel will boot and
operate normally inside a VM, however in this scenario it offers no benefits
compared to a standard Linux kernel.
We are investigating nested virtualization support to enable running okernel
inside VMs.


# Limitations and Caveats

The current implementation does not have any protection of the kernel
in place yet. It is a demonstration that you can create processes and
run them in NR-mode using EPTs with a shared kernel. As a further
demonstration of the concept, it implements protected memory pages,
whereby a process may request a protected memory page which will not
be mapped into the EPTs for other processes.

## Roadmap

The next step, and the subject of our ongoing research is to design
the memory protection architecture for the kernel. Examples of the
things that we are considering protecting from root mode processes
are:
 - Protection of the page tables (no NR mode process can modify a
   page table)
 - Protection of kernel executable code (RX only)
 - Protection of kernel data structures (RO)

# References

- [1] Nested Kernel: An Operating System Architecture for Intra-Kernel
Privilege Separation, Nathan Dautenhahn, Theodoros Kasampalis, Will
Dietz, John Criswell, Vikram Adve, ASPLOS '15, Proceedings of the
Twentieth International Conference on Architectural Support for
Programming Languages and Operating Systems, March 2015.
- [2] Dune: Safe user-level access to privileged CPU features, Adam
Belay, Andrea Bittau, Ali Mashtizadeh, David Terei, David Mazières,
and Christos Kozyrakis, OSDI '12, Proceedings of the 10th USENIX
Symposium on Operating Systems Design and Implementation, October 2012.
