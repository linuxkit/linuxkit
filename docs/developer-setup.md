# Build Platforms

This document describes how to install and maintain a LinuxKit development platform. It will grow over time.

The LinuxKit team also maintains several Linux-based build platforms. These are donated by Equinix Metal (arm64) and IBM (s390x).

## Platform-Specific Installation

### arm64 and amd64

The `amd64` and `arm64` platforms are fully supported by most OS vendors and Docker. Just upgrade to the latest OS and install the latest Docker using the
packaging tools. As of this writing, that is:

* Ubuntu/Debian with `apt`
* RHEL/CentOS/Fedora with `yum`. For any of these, use the CentOS 7/8 packages as released by Docker.

Docker does not recommend that you using the packages released by the OS vendors, as those tend to be out of date. Follow the instructions
[from Docker](https://docs.docker.com/engine/install/).

### s390x

The s390x has modern versions of most OSes, including RHEL and Ubuntu, but does not have recent versions of docker, neither as
`apt` packages for Ubuntu, nor as static downloads. In any case, these static downloads mostly are replicas.

This section describes how to install modern versions of Docker on these platforms.

#### RHEL

RHEL 7 on s390x only has releases from Docker. Follow the instructions from Docker to install. The rpm packages for RHEL are available at
https://download.docker.com/linux/rhel/

#### Ubuntu

Docker does not release packages for Ubuntu on s390x. The most recent release was for Ubuntu 18.04 Bionic, with Docker version 18.06.3.
This is quite old, and does not support modern capabilities, e.g. buildkit.

To install a more modern version:

1. Upgrade any dependent apt packages `apt upgrade`
1. Upgrade the operating system to your desired version `do-release-upgrade -d`. Note that you can set which versions to suggest via changing `/etc/update-manager/release-upgrades`
1. Download the necessary rpms (yes, rpms) from the Docker RHEL7 site. These are available [here](https://download.docker.com/linux/rhel/7/s390x/stable/Packages/). You need the following packages:
   * `containerd.io-*.rpm`
   * `docker-ce-*.rpm`
   * `docker-ce-cli-*.rpm`
1. Install alien: `apt install alien`
1. Convert each package to a dpkg `alien --scripts <source-rpm-file.rpm>`
1. Install each package with `dpkg -i <source-dpkg>.dpkg`. Dependency management is not great, so we recommend installing them in order:
   1. `containerd.io`
   1. `docker-ce`
   1. `docker-ce-cli`
1. Install devmapper `apt install libdevmapper-dev`
1. Check the missing version of libdevmapper, if any, with `ldd /usr/bin/dockerd`. In our example, it needs `libdevmapper.so.1.02`
1. Ensure that the library can be found where needed via `cd /lib/s390x-linux-gnu/ && ln -s $(ls -1 libdevmapper.so.*) libdevmapper.so.1.02`
1. Check again that dockerd is ok: `ldd /usr/bin/dockerd`
1. Start docker `system ctl restart docker`
1. Check that everything works:
   * `docker version`
   * `docker run --rm hello-world`

## Common Notes

On all platforms, if you want to run tests, you will need:

* `jq`
* `expect`
* `qemu-kvm`

These should be installed using your normal platform package installation, e.g. `apt install -y jq expect qemu-kvm`.

You also will need `rtf`, which can be installed with `make bin/rtf && make install`.

For pushing our kernels, you will need [manifest-tool](http://github.com/estesp/manifest-tool), which can be installed with
`make bin/manifest-tool && make install`.

Finally, to enable your regular user to run the tools, we recommend:

```
usermod -aG docker $USER
usermod -aG kvm $USER
usermod -aG sudo $USER
```
