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

## Preparation

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

Make sure that you have a recent version of the `linuxkit`
utility in the path. Either a previous release or compiled from
master.

On one of the build machines (preferably the `x86_64` machine), create
the branch:

```sh
git checkout -b $LK_BRANCH
```

Make sure that you have a recent version of the `linuxkit`
utility in the path. Either a previous release or compiled from
master.


### Update `linuxkit/alpine`

You must perform the arch-specific image builds, pushes and updates on each
architecture first - these can be done in parallel, if you choose. When done,
you then copy the updated `versions.<arch>` to one place, commit them, and
push the manifest.


#### Build and Push Per-Architecture

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

When all of the platforms are done, copy the changed `versions.<arch>` from each platform to one please, commit and push.
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

### Update tools packages

All of the `make push` and `make forcepush` in this section use `linuxkit pkg push`, which will build for all architectures and push
the images out. See [Build Platforms](./packages.md#Build_Platforms).

On your primary build machine, get the `linuxkit/alpine` tag and update the
other packages:

```sh
cd $LK_ROOT/tools
../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE
git checkout alpine/versions.aarch64 alpine/versions.s390x
git checkout grub/Dockerfile

git commit -a -s -m "tools: Update to the latest linuxkit/alpine"
git push $LK_REMOTE $LK_BRANCH

make forcepush
```

Note, the `git checkout` reverts the changes made by
`update-component-sha.sh` to files which are accidentally updated and
the `make forcepush` will skip building the alpine base.
Also, `git checkout` of `grub`. This is a bit old and only can be built with specific
older versions of packages like `gcc`, and should not be updated.

```sh
cd $LK_ROOT
for img in $(cd tools; make show-tag); do
    ./scripts/update-component-sha.sh --image $img
done

git commit -a -s -m "Update use of tools to latest"
```

### Update test packages

Next, we update the test packages to the updated alpine base:

```sh
cd $LK_ROOT/test/pkg
../../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "tests: Update packages to the latest linuxkit/alpine"
git push $LK_REMOTE $LK_BRANCH

make push
```

Next, update the use of test packages to latest:

```sh
cd $LK_ROOT
for img in $(cd test/pkg; make show-tag); do
    ./scripts/update-component-sha.sh --image $img
done

git commit -a -s -m "Update use of test packages to latest"
```

Some tests also use `linuxkit/alpine`. Update them as well:

```sh
cd $LK_ROOT/test/cases
../../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "tests: Update tests cases to the latest linuxkit/alpine"
```

### Update packages

Next, we update the LinuxKit packages. This is really the core of the
release. The other steps above are just there to ensure consistency
across packages.

```sh
cd $LK_ROOT/pkg
../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "pkgs: Update packages to the latest linuxkit/alpine"
git push $LK_REMOTE $LK_BRANCH
```

#### External Tools

Most of the packages are build from `linuxkit/alpine` and source code
in the `linuxkit` repository, but some packages wrap external
tools. When updating all packages, and especially during the time of a release,
is a good opportunity to check if there have been updates. Specifically:

- `pkg/cadvisor`: Check for [new releases](https://github.com/google/cadvisor/releases).
- `pkg/firmware` and `pkg/firmware-all`: Use latest commit from [here](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git).
- `pkg/node_exporter`: Check for [new releases](https://github.com/prometheus/node_exporter/releases).
- Check [docker hub](https://hub.docker.com/r/library/docker/tags/) for the latest `dind` tags. and update `examples/docker.yml`, `examples/docker-for-mac.yml`, `examples/cadvisor.yml`, and `test/cases/030_security/000_docker-bench/test.yml` if necessary.

Now build/push the packages and update the package tags in the YAML files. This step behaves
slightly differently, depending on whether you are just pushing out the images, or cutting a release.
The change in behaviour is determined by whether or not the environment variable `LK_RELEASE` is set.

Build and push out the packages:

```sh
cd $LK_ROOT/pkg
make push
```

Update the package tags:

```sh
cd $LK_ROOT
make update-package-tags
git commit -a -s -m "Update package tags"
```

Note that if you are cutting a release, you may want to change the above commit message
to include the release, for example:

```sh
git commit -a -s -m "Update package tags to $LK_RELEASE"
```

This is at your discretion.
