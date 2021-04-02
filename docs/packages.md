# LinuxKit packages

A LinuxKit package is a container image which can be used
to assemble a bootable Linux image. The LinuxKit project has a
number of [core packages](../pkg), but users can create their own
packages, as it's very easy. Packages are the unit of customisation
in a LinuxKit-based project, if you know how to build a container,
you should be able to build a LinuxKit package.

All official LinuxKit packages are:
- Enabled with multi-arch manifests to work on multiple architectures.
- Derived from well-known (and signed) sources for repeatable builds.
- Built with multi-stage builds to minimise their size.


## CI and Package Builds

When building and merging packages, it is important to note that our CI process builds packages. The targets `make ci` and `make ci-pr` execute `make -C pkg build`. These in turn execute `linuxkit pkg build` for each package under `pkg/`. This in turn will try to pull the image whose tag matches the tree hash or, failing that, to build it.

We do not want the builds to happen with each CI run for two reasons:

1. It is slower to do a package build than to just pull the latest image.
2. If any of the steps of the build fails, e.g. a `curl` download that depends on an intermittent target, it can cause all of CI to fail.

Thus, if, as a maintainer, you merge any commits into a `pkg/`, even if the change is documentation alone, please do a `linuxkit pkg push`.


## Package source

A package source consists of a directory containing at least two files:

- `build.yml`: contains metadata associated with the package
- `Dockerfile`: contains the steps to build the package.

`build.yml` contains the following fields:

- `image` _(string)_: *(mandatory)* The name of the image to build
- `org` _(string)_: The hub/registry organisation to which this package belongs
- `arches` _(list of string)_: The architectures which this package should be built for (valid entries are `GOARCH` names)
- `extra-sources` _(list of strings)_: Additional sources for the package outside the package directory. The format is `src:dst`, where `src` can be relative to the package directory and `dst` is the destination in the build context. This is useful for sharing files, such as vendored go code, between packages.
- `gitrepo` _(string)_: The git repository where the package source is kept.
- `network` _(bool)_: Allow network access during the package build (default: no)
- `disable-cache` _(bool)_: Disable build cache for this package (default: no)
- `config`: _(struct `github.com/moby/tool/src/moby.ImageConfig`)_: Image configuration, marshalled to JSON and added as `org.mobyproject.config` label on image (default: no label)
- `depends`: Contains information on prerequisites which must be satisfied in order to build the package. Has subfields:
    - `docker-images`: Docker images to be made available (as `tar` files via `docker image save`) within the package build context. Contains the following nested fields:
        - `from-file` and `list`: _(string and string list respectively)_. Mutually exclusive fields specifying the list of images to include. Each image must include a valid digest (`sha256:...`) in order to maintain determinism. If `from-file` is used then it is a path relative to (and within) the package directory with one image per line (lines with `#` in column 0 and blank lines are ignore). If `list` is used then each entry is an image.
        - `target` and `target-dir`: _(string)_ Mutually exclusive fields specifying the target location, if `target` is used then it is a path relative to (and within) the package dir which names a `tar` file into which all of the listed images will be saved. If `target-dir` then it is a path relative to (and within) the package directory which names a directory into which each image will be saved (as `«image name»@«digest».tar`). **NB**: The path referenced by `target-dir` will be _removed_ prior to populating (to avoid issues with stale files).

## Building packages

### Prerequisites

Before you can build packages you need:
- Docker version 19.03 or newer, which includes [buildx](https://docs.docker.com/buildx/working-with-buildx/)
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

LinuxKit builds packages as docker images. It deposits the built package as a docker image in one of two targets:

* the linuxkit cache `~/.linuxkit/` (configurable) - default option
* the docker image cache

If you want to build images and test and run them _in a standalone_ fashion locally, then you should pick the docker image cache. Otherwise, you should use the default linuxkit cache. LinuxKit defaults to building OS images using docker images from this cache,\
only looking in the docker cache if instructed to via `linuxkit build --docker`.

When using the linuxkit cache as the package build target, it creates all of the layers, the manifest that can be uploaded
to a registry, and the multi-architecture index. If an image already exists for a different architecture in the cache,
it updates the index to include additional manifests created.

As of this writing, `linuxkit pkg build` only builds packages for the platform on which it is running; it does not (yet) support cross-building the packages for other architectures.

Note that the local docker option is available _only_ when building without pushing to a remote registry, i.e.:

```
linuxkit pkg build
linuxkit pkg build --docker
```

If you push to a registry, it _always_ uses the linuxkit cache only:

```
linuxkit pkg push
```

### Build packages as a maintainer

All official LinuxKit packages are multi-arch manifests and most of
them are available for the following platforms:

* `linux/amd64`
* `linux/arm64`
* `linux/s390x`

Official images *must* be built on all architectures for which they are available.
They can be built and pushed in parallel, but the manifest should be pushed once
when all of the images are done.

Pushing out a package as a maintainer involves two distinct stages:

1. Building and pushing out the platform-specific image
1. Creating, pushing out and signing the multi-arch manifest, a.k.a. OCI image index

The `linuxkit pkg` command contains automation which performs all of the steps.
Note that `«path-to-package»` is the path to the package's source directory
(containing at least `build.yml` and `Dockerfile`). It can be `.` if
the package is in the current directory.


#### Image Only

To build and push out the platform-specific image, on that platform:

```
linuxkit pkg push --manifest=false «path-to-package»
```

The options do the following:

* `--manifest=false` means not to push or sign a manifest

Repeat the above on each platform where you need an image.

This will do the following:

1. Determine the name and tag for the image as follows:
   * The tag is from the hash of the git tree for that package. You can see it by doing `linuxkit pkg show-tag «path-to-package»`.
   * The name for the image is from `«path-to-package»/build.yml`
   * The organization for the package is given on the command-line, default to `linuxkit`.
1. Build the package in the given path using your local docker instance for the local platform. E.g. if you are running on `linux/arm64`, it will build for `linux/arm64`.
1. Tag the build image as `«image-name»:«hash»-«arch»`
1. Push the image to the hub

#### Manifest Only

To perform just the manifest steps, do:

```
linuxkit pkg push --image=false --manifest «path-to-package»
```

The options do the following:

* `--image=false` do not push the image, as you already did it; you can, of course, skip this argument and push the image as well
* `--manifest` create and push the manifest

This will do the following:

1. Find all of the images on the hub of the format `«image-name»:«hash»-«arch»`
1. Create a multi-arch manifest called `«image-name»:«hash»` (note no `-«arch»`)
1. Push the manifest to the hub
1. Sign the manifest with your key

Each time you perform the manifest steps, it will find all of the images,
including any that have been added since last time.
The LinuxKit YAML files should consume the package as the multi-arch manifest:
`linuxkit/<image>:<hash>`.

#### Everything at once

To perform _all_ of the steps at once - build and push out the image for whatever platform
you are running on, and create and sign a manifest - do:

```
linuxkit pkg push «path-to-package»
```

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
