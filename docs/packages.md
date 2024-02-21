# LinuxKit packages

A LinuxKit package is a container image which can be used
to assemble a bootable Linux image. The LinuxKit project has a
number of [core packages](../pkg), but users can create their own
packages, as it's very easy. Packages are the unit of customisation
in a LinuxKit-based project, if you know how to build a container,
you should be able to build a LinuxKit package.

All official LinuxKit packages are:
- Enabled with multi-arch indexes to work on multiple architectures.
- Derived from well-known sources for repeatable builds.
- Built with multi-stage builds to minimise their size.


## CI and Package Builds

When building and merging packages, it is important to note that our CI process builds packages. The targets `make ci` and `make ci-pr` execute `make -C pkg build`. These in turn execute `linuxkit pkg build` for each package under `pkg/`. This in turn will try to pull the image whose tag matches the tree hash or, failing that, to build it.

Any released image, i.e. any package under `pkg/` that has _not_ changed as
part of a pull request,
already will be released to Docker Hub. This will cause it to download that image, rather
than try to build it.

Any non-releaed image, i.e. any package under `pkg/` that _has_ changed as part of
a pull request, will not be in Docker Hub until the PR has merged.
This will cause the download to fail, leading `linuxkit pkg build` to try and build the
image and save it in the cache.

This does have two downsides:

1. It is slower to do a package build than to just pull the latest image.
2. If any of the steps of the build fails, e.g. a `curl` download that depends on an intermittent target, it can cause all of CI to fail.

In the past, each PR required a maintainer to build, and push to Docker Hub, every
changed package in `pkg/`. This placed the maintainer in the PR cycle, with the
following downsides:

1. A maintainer had to be involved in every PR, not just reviewing but actually building and pushing. This reduces the ability for others to contribute.
1. The actual package is pushed out by a person, violating good supply-chain practice.

## Package source

A package source consists of a directory containing at least two files:

- `build.yml`: contains metadata associated with the package
- `Dockerfile`: contains the steps to build the package.

`build.yml` contains the following fields:

- `image` _(string)_: *(mandatory)* The name of the image to build
- `org` _(string)_: The hub/registry organisation to which this package belongs
- `dockerfile` _(string)_: The dockerfile to use to build this package, must be in this directory or below (default: `Dockerfile`)
- `arches` _(list of string)_: The architectures which this package should be built for (valid entries are `GOARCH` names)
- `extra-sources` _(list of strings)_: Additional sources for the package outside the package directory. The format is `src:dst`, where `src` can be relative to the package directory and `dst` is the destination in the build context. This is useful for sharing files, such as vendored go code, between packages.
- `gitrepo` _(string)_: The git repository where the package source is kept.
- `network` _(bool)_: Allow network access during the package build (default: no)
- `disable-cache` _(bool)_: Disable build cache for this package (default: no)
- `buildArgs` will forward a list of build arguments down to docker. As if `--build-arg` was specified during `docker build`
- `config`: _(struct `github.com/moby/tool/src/moby.ImageConfig`)_: Image configuration, marshalled to JSON and added as `org.mobyproject.config` label on image (default: no label)
- `depends`: Contains information on prerequisites which must be satisfied in order to build the package. Has subfields:
    - `docker-images`: Docker images to be made available (as `tar` files via `docker image save`) within the package build context. Contains the following nested fields:
        - `from-file` and `list`: _(string and string list respectively)_. Mutually exclusive fields specifying the list of images to include. Each image must include a valid digest (`sha256:...`) in order to maintain determinism. If `from-file` is used then it is a path relative to (and within) the package directory with one image per line (lines with `#` in column 0 and blank lines are ignore). If `list` is used then each entry is an image.
        - `target` and `target-dir`: _(string)_ Mutually exclusive fields specifying the target location, if `target` is used then it is a path relative to (and within) the package dir which names a `tar` file into which all of the listed images will be saved. If `target-dir` then it is a path relative to (and within) the package directory which names a directory into which each image will be saved (as `«image name»@«digest».tar`). **NB**: The path referenced by `target-dir` will be _removed_ prior to populating (to avoid issues with stale files).

## Building packages

### Prerequisites

Before you can build packages you need:
- Docker version 19.03 or newer.
- If you are on a Mac you also need `docker-credential-osxkeychain.bin`, which comes with Docker for Mac.
- `make`, `base64`, `jq`, and `expect`
- A *recent* version of `manifest-tool` which you can build with `make
  bin/manifest-tool`, or `go get github.com:estesp/manifest-tool`, or
  via the LinuxKit homebrew tap with `brew install --HEAD
  manifest-tool`. `manifest-tool` must be in your path.
- The LinuxKit tool `linuxkit` which must be in your path.

Further, when building packages you need to be logged into hub with
`docker login` as some of the tooling extracts your hub credentials
during the build.

### Build Targets

LinuxKit builds packages as docker images. It deposits the built package as a docker image in one or both of two targets:

* the linuxkit cache, which is at `~/.linuxkit/cache/` (configurable)
* the docker image cache (optional)

The package _always_ is built and saved in the linuxkit cache. However, you _also_ can load the package for the current
architecture, if available, into the docker image cache.

If you want to build images and test and run them _in a standalone_ fashion locally, then you should add the docker image cache.
Otherwise, you don't need anything more than the default linuxkit cache. LinuxKit defaults to building OS images using docker
images from this cache, only looking in the docker cache if instructed to via `linuxkit build --docker`.

In the linuxkit cache, it creates all of the layers, the manifest that can be uploaded
to a registry, and the multi-architecture index. If an image already exists for a different architecture in the cache,
it updates the index to include additional manifests created.

The order of building is as follows:

1. Build the image to the linuxkit cache
1. If `--docker` is provided, load the image into the docker image cache

For example:

```bash
linuxkit pkg build pkg/foo           # builds pkg/foo and places it in the linuxkit cache
linuxkit pkg build pkg/foo --docker  # builds pkg/foo and places it in the linuxkit cache and also loads it into docker
```

#### Build Platforms

By default, `linuxkit pkg build` builds for all supported platforms in the package's `build.yml`, whose syntax is available
[here][Package source]. If no platforms are provided in the `build.yml`, it builds for all platforms that linuxkit supports.
As of this writing, those are:

* `linux/amd64`
* `linux/arm64`
* `linux/s390x`

You can choose to skip one of the platforms from `build.yml` or those selected
by default using the `--skip-platforms` flag.

For example:

```
linuxkit pkg build --skip-platforms linux/s390x ...
```

You can override the target build platform by passing it the `--platforms` option:

```
linuxkit pkg build --platforms <platform1,platform2,...platformN>
```

The options for `--platforms` are identical to those for [docker build](https://docs.docker.com/engine/reference/commandline/build/).
An example is available in the official [buildx documentation](https://docs.docker.com/buildx/working-with-buildx/#build-multi-platform-images).

Given that this is linuxkit, i.e. all builds are for linux, the `OS` part would seem redundant, and it should be sufficient to pass `--platform arm64`. However, for complete consistency, the _entire_ platform, e.g. `--platforms linux/amd64,linux/arm64`, must be provided.

#### Where it builds

You are running the `linuxkit pkg build` command on a single platform, e.g. your local linux cloud instance running on `amd64`, or
a MacBook with Apple Silicon running on `arm64`.

How does linuxkit determine where to build the target images?

linuxkit uses [buildkit](https://github.com/moby/buildkit) directly to build all images.
It uses docker contexts to determine _where_ to run those buildkit containers, based on the target
architecture.

When running a package build, linuxkit looks for a container named `linuxkit-builder`, running the appropriate
version of buildkit. If it cannot find a container with that name, it creates it.
If the container already exists but is not running buildkit, or if the version is incorrect, linuxkit stops and removes
the existing `linuxkit-builder` container and creates one running the correct version of buildkit.

When linuxkit needs to build a package for a particular architecture:

1. If a context for that architecture was provided, use that context, looking for and/or starting a buildkit container named `linuxkit-builder`.
1. If no context for that architecture was provided, use the `default` context.

The actual building then will be one of:

1. native, if the provided context has the same architecture as the target build architecture; else
1. cross-build, if the provided context has a different architecture, but the package's `Dockerfile` supports cross-building; else
1. emulated build, using docker's qemu binfmt capabilities

Cross-building, i.e. building on one platform using that platform's binaries to create outputs for a different platform,
depends on the package's `Dockerfile`. Details are available in the
[official Docker buildx docs](https://docs.docker.com/buildx/working-with-buildx/#build-multi-platform-images).

* if the image is just `FROM something`, then it runs it under qemu using binfmt
* if the image is `FROM --platform=$BUILDPLATFORM something`, then it runs it using the local architecture, invoking cross-builders

Read the official docs to learn more how to leverage cross-building with buildx.

**Important:** When building, if the local architecture is not one of those being build,
selecting `--docker` to load the images into the docker image cache will result in an error.
You _must_ be building for the local architecture - optionally for others as well - in order to
pass the `--docker` option.

#### Providing native builder nodes

linuxkit is capable of using native build nodes to do the build, even remotely. To do so, you must:

1. Create a [docker context](https://docs.docker.com/engine/context/working-with-contexts/) that references the build node
1. Tell linuxkit to use that context for that architecture

linuxkit will then use that provided context to look for and/or start a container in which to run buildkit for that architecture.

linuxkit looks for contexts in the following descending order of priority:

1. CLI option `--builders <platform>=<context>,<platform>=<context>`, e.g. `--builders linux/arm64=linuxkit-arm64,linux/amd64=default`
1. Environment variable `LINUXKIT_BUILDERS=<platform>=<context>,<platform>=<context>`, e.g. `LINUXKIT_BUILDERS=linux/arm64=linuxkit-arm64,linux/amd64=default`
1. Existing context named `linuxkit-<platform>`, e.g. `linuxkit-linux-arm64` or `linuxkit-linux-s390x`, with "/" replaced by "-", as "/" is an invalid character.
1. Default context

If a builder name is provided for a specific platform, and it doesn't exist, it will be treated as a fatal error.

#### Examples

##### Simple build

There are no contexts starting with `linuxkit-`, no environment variable `LINUXKIT_BUILDERS`, no command-line argument `--builders`.

linuxkit will build any requested packages using `default` context on the local platform, with a container (created, if necessary) named `linuxkit-builder`.
Builds for the same architecture will be native, builds for other platforms will use either qemu or cross-building.

##### Specified target

You create a context named `my-remote-arm64` and then run:

```bash
linuxkit pkg build --platforms=linux/arm64,linux/amd64 --builders linux/arm64=my-remote-arm64
```

linuxkit will build:

* for arm64 using the context `my-remote-arm64`, since you specified in `--builders` to use `my-remote-arm64` for `linux/arm64`
* for amd64 using the context `default`, as that is the default fallback

The same would happen if you used `LINUXKIT_BUILDERS=linux/arm64=my-remote-arm64` instead of the `--builders` flag.

In both cases - the remote context `my-remote-arm64` and the local `default` context - it will do the build inside
a container named `linuxkit-builder`.

##### Named context

You create a context named `linuxkit-linux-arm64` and then run:

```bash
linuxkit pkg build --platforms=linux/arm64,linux/amd64
```

linuxkit will build:

* for arm64 using the context `linuxkit-linux-arm64`, since there is a context with the name `linuxkit-<platform>`, and you did not override it using `--builders` or the environment variable `LINUXKIT_BUILDERS`
* for amd64 using the context `default` and the `linuxkit` builder, as that is the default fallback

##### Combination

You create a context named `linuxkit-linux-arm64`, and another named `my-remote-builder-amd64` and then run:

```bash
linuxkit pkg build --platforms=linux/arm64,linux/amd64 --builders linux/amd64=my-remote-builder-amd64
```

linuxkit will build:

* for arm64 using the context `linuxkit-linux-arm64`, since there is a context with the name `linuxkit-<platform>`, and you did not override that particular architecture using `--builders` or the environment variable `LINUXKIT_BUILDERS`
* for amd64 using the context `my-remote-builder-amd64`, since you specified for that architecture using `--builders`

The same would happen if you used `LINUXKIT_BUILDERS=linux/arm64=my-remote-builder-amd64` instead of the `--builders` flag.

##### Missing context

You do not have a context named `my-remote-arm64`, and run:

```bash
linuxkit pkg build --platforms=linux/arm64 --builders linux/arm64=my-remote-arm64
```

linuxkit will try to build for `linux/arm64` using the context `my-remote-arm64`. Since that context does not exist, you will get an error.

##### Preset build arguments

When building packages, the following build-args automatically are set for you:

* `SOURCE` - the source repository of the package
* `REVISION` - the git commit that was used for the build
* `GOPKGVERSION` - the go package version or pseudo-version per https://go.dev/ref/mod#glos-pseudo-version

Note that the above are set **only** if you do not set them in `build.yaml`. Your settings _always_
override these built-in ones.

To use them, simply address them in your `Dockerfile`:

```dockerfile
ARG SOURCE
```

### Build packages as a maintainer

All official LinuxKit packages are multi-arch manifests and most of
them are available for the following platforms:

* `linux/amd64`
* `linux/arm64`
* `linux/s390x`

Official images *must* be built for all architectures for which they are available.

Pushing out a package as a maintainer involves two stages:

1. Building and pushing out the platform-specific images
1. Creating and pushing out the multi-arch manifest, a.k.a. OCI image index

The `linuxkit pkg` command contains automation which performs all of the steps.
Note that `«path-to-package»` is the path to the package's source directory
(containing at least `build.yml` and `Dockerfile`). It can be `.` if
the package is in the current directory.

```
linuxkit pkg push «path-to-package»
```

This will do the following:

1. Determine the name and tag for the image as follows:
   * The tag is from the hash of the git tree for that package. You can see it by doing `linuxkit pkg show-tag «path-to-package»`.
   * The name for the image is from `«path-to-package»/build.yml`
   * The organization for the package is given on the command-line, default to `linuxkit`.
1. Build the package in the given path using your local docker instance for all the platforms in `«path-to-package»/build.yml`
1. Save the built image in the linuxkit cache
1. Tag each built image as `«image-name»:«hash»-«arch»`
1. Create a multi-arch manifest called `«image-name»:«hash»` (note no `-«arch»`)
1. Push the manifest and all of the images to the hub

Note that for actual release images, these steps normally are performed as part
of CI, by the merge-to-master process.

#### Prerequisites

* For all of the steps, you *must* be logged into hub (`docker login`).

### Build packages as a developer


```
linuxkit pkg build -org=wombat «path-to-package»
```

This will create a local image: `wombat/<image>:<hash>-<arch>` which
you can use in your local YAML files for testing. If you need to test
on other systems you can push the image to your hub account and pull
from a different system by issuing:

```
linuxkit pkg build -org=wombat push
```

This will push both `wombat/<image>:<hash>-<arch>` and
`wombat/<image>:<hash>` to hub.

Finally, if you are tired of the long hashes you can override the hash
with:

```
linuxkit pkg build -org=wombat -hash=foo push
```

and this will create `wombat/<image>:foo-<arch>` and
`wombat/<image>:foo` for use in your YAML files.

### Proxies

If you are building packages from behind a proxy, `linuxkit pkg build` respects
the following environment variables, and will set them as `--build-arg` to
`docker build` when building a package.

* `http_proxy` / `HTTP_PROXY`
* `https_proxy` / `HTTPS_PROXY`
* `ftp_proxy` / `FTP_PROXY`
* `no_proxy` / `NO_PROXY`
* `all_proxy` / `ALL_PROXY`

Note that the first four of these are the standard built-in `build-arg` options available
for `docker build`; see the [docker build documentation](https://docs.docker.com/v17.09/engine/reference/builder/#arg).
The last, `all_proxy`, is a standard var used for socks proxying. Since it is not built into `docker build`,
if you want to use it, you will need to add the following line to the dockerfile:

```dockerfile
ARG all_proxy
```

LinuxKit does not judge between lower-cased or upper-cased variants of these options, e.g. `http_proxy` vs `HTTP_PROXY`,
as `docker build` does not either. It just passes them through "as-is".
