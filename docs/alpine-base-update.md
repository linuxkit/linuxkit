# Updating Alpine Base

This document describes the steps to update the `linuxkit/alpine` image.
This image is at the base of all other linuxkit images.
It is built out of the directory `tools/alpine/`.

While you do not need to update every downstream image _immediately_ when you update
this image, you do need to be aware that changes to this image will affect the
downstream images when it is next adopted. Those downstream images should be updated
as soon as possible after updating `linuxkit/alpine`.

When you make a linuxkit release, you _must_ update all of the downstream images.
See [releasing.md](./releasing.md) for the release process.

## Pre-requisites

Updating `linuxkit/alpine` can be done by any maintainer. Maintainers need to have
access to build machines for all architectures support by LinuxKit.

## Process

At a high-level, we are going to do the following:

1. Preparatory steps
1. Create a new branch
1. Make our desired changes to `tools/alpine` and commit them
1. Build and push out our alpine changes, and commit the `versions` files
1. Update all affected downstream changes and commit them: `tools/`, `test/pkg`, `pkg`, `test/`, `examples/`
1. Push out all affected downstream changes: `tools/`, `test/pkg`, `pkg`, `test/`, `examples/`

For each of the affected downstream changes, we could update and then push, then move to the next. However,
since the push out can be slow and require retries, we try to make all of the changes first, and then push them out.

### Preparation

As a starting point you have to be on the update to date master branch
and be in the root directory of your local git clone. You should also
have the same setup on all build machines used.

To make the steps below cut-and-pastable, define the following
environment variables:

```sh
LK_ROOT=$(pwd)
LK_REMOTE=origin         # or whatever your personal remote is
LK_BRANCH=alpine-update  # or whatever the name of the branch on which you are working is
```

Note that if you are cutting a release, the `LK_BRANCH` may have a release-type name, e.g. `rel_v0.4`.

Make sure that you have the latest version of the `linuxkit`
utility in the path. Alternatively, you may wish to compile the latest version from
master.

### Create a new branch

On one of the build machines (preferably the `x86_64` machine), create
the branch:

```sh
git checkout -b $LK_BRANCH
```

### Update `linuxkit/alpine`

You must perform the arch-specific image builds, pushes and updates on each
architecture first - these can be done in parallel, if you choose. When done,
you then copy the updated `versions.<arch>` to one place, commit them, and
push the manifest.

#### Make alpine changes

Make any changes in `tools/alpine` that you desire, then commit them.
In the below, change the commit message to something meaningful to the change you are making.

```sh
cd tools/alpine
# make changes
git commit -s -a -m "Update linuxkit/alpine"
git push origin $LK_BRANCH
```

#### Build and Push Alpine Per-Architecture

On each supported platform, build and update `linuxkit/alpine`, which will update the `versions.<arch>`
file.:

```sh
git fetch
git checkout $LK_BRANCH
cd $LK_ROOT/tools/alpine
make push
```

Repeat on each platform.

#### Commit Changed Versions Files

When all of the platforms are done, copy the changed `versions.<arch>` from each platform to one place, commit and push.
In the below, replace `linuxkit-arch` with each build machine's name:

```sh
# one of these will not be necessary, as you will likely be executing it on one of these machines
scp linuxkit-s390x:$LK_ROOT/tools/alpine/versions.s390x $LK_ROOT/tools/alpine/versions.s390x
scp linuxkit-aarch64:$LK_ROOT/tools/alpine/versions.aarch64 $LK_ROOT/tools/alpine/versions.aarch64
scp linuxkit-x86_64:$LK_ROOT/tools/alpine/versions.x86_64 $LK_ROOT/tools/alpine/versions.x86_64
git commit -a -s -m "tools/alpine: Update to latest"
git push $LK_REMOTE $LK_BRANCH
```

#### Update and Push Multi-Arch Index

Push out the multi-arch index:

```sh
make push-manifest
```

Stash the tag of the alpine base image in an environment variable:

```sh
LK_ALPINE=$(make show-tag)
```

### Update affected downstream packages 

This section describes all of the steps. Below follows a straight copyable list of steps to take,
following which is an explanation of each one.

```sh
# Update tools packages
cd $LK_ROOT/tools
$LK_ROOT/scripts/update-component-sha.sh --image $LK_ALPINE
git checkout grub-dev/Dockerfile
git checkout mkimage-rpi3/Dockerfile
git commit -a -s -m "tools: Update to the latest linuxkit/alpine"

# Update tools dependencies
cd $LK_ROOT
for img in $(cd tools; make show-tag); do
    $LK_ROOT/scripts/update-component-sha.sh --image $img
done
git commit -a -s -m "Update use of tools to latest"

# Update test packages
cd $LK_ROOT/test/pkg
$LK_ROOT/scripts/update-component-sha.sh --image $LK_ALPINE
git commit -a -s -m "tests: Update packages to the latest linuxkit/alpine"

# Update test packages dependencies
cd $LK_ROOT
for img in $(cd test/pkg; make show-tag); do
    $LK_ROOT/scripts/update-component-sha.sh --image $img
done
git commit -a -s -m "Update use of test packages to latest"

# Update test cases to latest linuxkit/alpine
cd $LK_ROOT/test/cases
$LK_ROOT/scripts/update-component-sha.sh --image $LK_ALPINE
git commit -a -s -m "tests: Update tests cases to the latest linuxkit/alpine"

# Update packages to latest linuxkit/alpine
cd $LK_ROOT/pkg
$LK_ROOT/scripts/update-component-sha.sh --image $LK_ALPINE
git commit -a -s -m "pkgs: Update packages to the latest linuxkit/alpine"

# update package tags - may want to include the release in it if set
cd $LK_ROOT
make update-package-tags
MSG=""
[ -n "$LK_RELEASE" ] && MSG="to $LK_RELEASE"
git commit -a -s -m "Update package tags $MSG"

git push $LK_REMOTE $LK_BRANCH
```

#### Update tools packages

On your primary build machine, update the other tools packages.

Note, the `git checkout` reverts the changes made by
`update-component-sha.sh` to files which are accidentally updated.
Important is the `git checkout` of some sensitive packages that only can be built with
specific older versions of upstream packages:

* `grub-dev`
* `mkimage-rpi3`

Only update those if you know what you are doing with them.

Then we update any dependencies of these tools.

#### Update test packages

Next, we update the test packages to the updated alpine base.

Next, we update the use of test packages to latest.

Some tests also use `linuxkit/alpine`, so we update them as well.

### Update packages

Next, we update the LinuxKit packages. This is really the core of the
release. The other steps above are just there to ensure consistency
across packages.

#### External Tools

Most of the packages are build from `linuxkit/alpine` and source code
in the `linuxkit` repository, but some packages wrap external
tools. When updating all packages, and especially during the time of a release,
is a good opportunity to check if there have been updates. Specifically:

- `pkg/cadvisor`: Check for [new releases](https://github.com/google/cadvisor/releases).
- `pkg/firmware` and `pkg/firmware-all`: Use latest commit from [here](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git).
- `pkg/node_exporter`: Check for [new releases](https://github.com/prometheus/node_exporter/releases).
- Check [docker hub](https://hub.docker.com/r/library/docker/tags/) for the latest `dind` tags. and update `examples/docker.yml`, `examples/docker-for-mac.yml`, `examples/cadvisor.yml`, and `test/cases/030_security/000_docker-bench/test.yml` if necessary.

This is at your discretion.

### Build and push affected downstream packages

<ul>Note</ul>: All of the `make push` and `make forcepush` in this section use `linuxkit pkg push`, which will build for all architectures and push
the images out. See [Build Platforms](./packages.md#Build_Platforms).

```sh
# build and push out the tools packages
cd $LK_ROOT/tools
make forcepush

# Build and push out test packages
cd $LK_ROOT/test/pkg
make push

# build and push out the packages
cd $LK_ROOT/pkg
make push
```
