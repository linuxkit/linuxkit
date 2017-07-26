# Trusted Computing

LinuxKit has support for using [Trusted Computing](http://trustedcomputinggroup.org) Platform Modules (tpm) chips.

Supporting tpm requires support at three levels:

* Hardware
* Kernel
* Software - The Trusted Computing Software Stack (TSS)

## Hardware
You need to have a tpm chip installed in your computer to use tpm. Alternatively, you can use one of the virtual tpms implemented in software, provided that either:

* your kernel supports it
* your hardware virtualization platform supports it

## Kernel
As of [PR 2234](https://github.com/linuxkit/linuxkit/pull/2234), the in-tree linux kernel modules that support tpm are shipped with LinuxKit by default.

The shipped modules support both tpm chip versions 1.2 and tpm 2.0.

## Software
The software stack (TSS) functions differently between tpm versions 1.2 and 2.0.

### tss 1.2
In tss 1.2, the character device `/dev/tpm0` is meant to be addresses only by a single process. All other clients are expected to communicate with this single client that handles multiplexing of requests and various other low-level functionality.

The single client normally used is [TrouSerS](https://sourceforge.net/p/trousers/trousers/). It creates a daemon, `tcsd`, that communicates with the character device (and via the character device and the kernel module to the actual tpm).

`tcsd` in turn listens on `localhost:30003` for tpm commands. All other clients are expected to communicate via tcp to `tcsd`.

LinuxKit provides the `linuxkit/tss` image which includes:

* `tcsd`
* the various `tpm_*` tools

To make a `tcsd` available to your LinuxKit image, just include it:

```yml
services:
  - name: tss
    image: linuxkit/tss:<hash>
```

For a full example, see [tpm.yml](../examples/tpm.yml)

### tss 2.0
In tss 2.0, the character device `/dev/tpmrm0` can be addressed by as many processes, in parallel, as desired. All of the multiplexing and low-level services are built into the kernel module.

To use a tpm 2.0 device, you do **not** need any special tss container. You just need an container that:

1. Bind-mounts `/dev` in
2. Has your tools or libraries installed
3. Talks directly to `/dev/tpmrm0`


The image `linuxkit/tss` ships with the version 1.2 `tcsd` and the `tpm_*` tools for tpm version 1.2. The tools for tpm version 2.0 `tpm2_*` and its attendant libs are _not_ included in the image.

We intend to release a tss 2.0 compatible image in the near future. In the meantime, nothing prevents you from using and compiling your own tss and including it in a LinuxKit image.
