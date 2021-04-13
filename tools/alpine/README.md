# LinuxKit Alpine

`linuxkit/alpine` is the base image for almost all other packages built by linuxkit, including builders, tools and actual container images
that are used in various parts of linuxkit yaml files.

This provides a reliable, consistent and repetable build.

This directory contains the source of `linuxkit/alpine`.

## Building

To build, run:

```
make build
```

## Pushing

To push, run:

```
make push
```

For a proper release process, see [docs/releasing.md](../../docs/releasing.md).

## Updating Sources and Packages

The base build for `linuxkit/alpine` is [library/alpine](https://hub.docker.io/_/alpine). The specific version is set in two `FROM` lines in
the [Dockerfile](./Dockerfile) in this directory.

The packages installed come from several sources:

* [packages](./packages) - this file contains the list of packages to mirror locally in `linuxkit/alpine`, and will be available to all downstream users of `linuxkit/alpine`. These are installed using the default `apk` package version for the specific version of alpine. For example, if the line starts with `FROM alpine:3.13` and `packages` contains `file`, then it will run simply `apk add file`. The packages listed in [packages](./packages) are installed on all architectures.
* `packages.<arch>` - these files contain the list of packages to mirror locally in `linuxkit/alpine`, like `packages`, but only for the specified architecture. For example, [packages.x86_64](./packages.x86_64) contains packages to be installed only on `linuxkit/alpine` for `x84_64`.
* `packages.repo.<name>` - these files contain the list of packages to mirror locally in `linuxkit/alpine`, like `packages`, but to pull those packages from the provided `<name>` of Alpine's `apk` repo. For example, `packages.repo.edge` installs packages from Alpine's `edge` package repository.
* `packages.<arch>.repo.<name>` - these files contain the list of packages to mirror locally in `linuxkit/alpine` for a specific architecture, like `packages.<arch>`, but to pull those packages from the provided `<name>` of Alpine's `apk` repor. For example, `packages.x86_64.repo.edge` installs packages from Alpine's `edge` package repository, hut only when building for `86_64`.

In addition, the [Dockerfile](./Dockerfile) may install certain packages directly from source, if they are not available in the `apk` repositories, or the versions are
insufficient.

The final versions of packages installed are available in `versions.<arch>`.
