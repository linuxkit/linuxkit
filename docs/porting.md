Porting Linuxlit to new architectures
=====================================

Linuxkit can be logcially seperated in to 3 areas, in order of increasing maintenance cost:

1. The Build Tool
1. The Packages
1. The Kernel

We can collectively refer to the packages and kernel as the "content library" as this is what we provide via Docker Hub.

An architecture with first class support will have support in the build tool, and the necessary plumbing and machines to publish images in our content library. Staying within these parameters is the easiest option for `lkt` users.

However, one of the promises of Linuxkit is that you can bring your own packages and kernels.
Which means that the community of users can easily make progress on a port to a new architecture.

## Path to conbtribution upstream

It's recommended that you engage the maintainers early. File a GitHub issue, join the Slack channel
and communitcate your intent! We'll do our best to help you with the process and give you a path
to get the work merged upstream.

Progress to first class support happens in 3 stages.
During each of these stages you can expect limited support from the linuxkit community, which is outlined below.

### The Build Tool

`lkt pkg build` should work for compiling packages an any architecture since it supports binfmt_misc based cross compilation and using remote builders.

**Support**
:heavy_check_mark: The build tool used is still supported by the linuxkit community
:heavy_check_mark: Contributions to make the tool more useful are always welcome

### The Packages

The requirements for a new architecture are that:
1. It must be supported in Alpine
1. There must be Official Docker Images available for that architecture
1. A new linuxkit/alpine base image must be built and added to the multi-arch manifest

In the event that Alpine doesn't support the architecture you'll need to engage the Alpine Linux Community.
In the event that Official Docker Images aren't available, you'll need to engage the Docker Library maintainers.

It's possible for the community to make progess by simply switching out the base image from the supported one.
This is likely to expose a lot of issues (i.e missing packages, packages that don't compile etc...) where generic
fixes could be accepted by the various upstream projects.

**Graduation requirements**
- The library of packages in `pkg` that support `all` architectures can be built and pushed for the new architecture
- All package tests for the new architecture succeed
- The `examples` can be built and run on a new architecture

**Support**
:heavy_check_mark: The package code is still supported by the linuxkit community
:heavy_check_mark: Contributions to make these packages work for all arches are welcome
:x: Any content published outside the linuxkit org on hub is not supported by the linuxkit community

### The Kernel

Instructions for building kernels can be found in [here](docs/kernels.md).

**Graduation requirements**
- A kernel can be built using the scripts in this repo.
- That kernel can be used with `lkt build` to boot a system.
- The kernel passes the kernel test suite.

**Support**
:heavy_check_mark: The kernel build scripts are still supported by the linuxkit community
:heavy_check_mark: Contributions to make this emit useable kernels work for all arches are welcome
:x: Any kernel published outside the linuxkit org on hub is not supported by the linuxkit community

### Final Checks

Before adding the arch as one of the first-class supported ones we must ensure that:

1. There is documentation that explains how to use it!
1. We have a working CI setup including access to machines for build/push of images
1. We have either:
  a) A commitment from a maintainer that they are willing to maintain this
  b) A volunteer to join the motley crew of maintainers who is willing to maintain this
