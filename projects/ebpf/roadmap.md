# iovisor eBPF tools

The [iovisor eBPF tools](https://github.com/iovisor/bcc) are a
collection tools useful for debugging and performance analysis. This
project aims to provide an easy to consume packaging of these tools
for Moby.

It comes in two parts: [ebpf.build](ebpf.build/) is used for building the binaries on alpine and [ebpf.pkg](ebpf.pkg/) is supposed to package the binaries with the required kernel files.

**Note**: The packages currently do not build.

## Roadmap

**Near-term:**
- Make the package build again

**Mid-term:**
- Trim the distribution package to make it easier to consume
