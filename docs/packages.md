# LinuxKit packages

A LinuxKit package is a container image which can be used
to assemble a bootable Linux image. The LinuxKit project has a
number of [core packages](../pkg), but users can create their own
packages, as it's very easy. Packages are the unit of customisation
in a LinuxKit-based project, if you know how to build a container,
you should be able to build a LinuxKit package.

All official LinuxKit packages are:
- Signed with Docker Content Trust.
- Enabled with multi-arch manifests to work on multiple architectures.
- Derived from well-known (and signed) sources for repeatable builds.
- Built with multi-stage builds to minimise their size.


## CI and Package Builds

When building and merging packages, it is important to note that our CI process builds packages. The targets `make ci` and `make ci-pr` execute `make -C pkg build`. These in turn execute `linuxkit pkg build` for each package under `pkg/`. This in turn will try to pull the image whose tag matches the tree hash or, failing that, to build it.

We do not want the builds to happen with each CI run for two reasons:

1. It is slower to do a package build than to just pull the latest image.
2. If any of the steps of the build fails, e.g. a `curl` download that depends on an intermittent target, it can cause all of CI to fail.

Thus, if, as a maintainer, you merge any commits into a `pkg/`, even if the change is documentation alone, please do a `linuxkit package push`.


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
- `disable-content-trust` _(bool)_: Disable Docker content trust for this package (default: no)
- `disable-cache` _(bool)_: Disable build cache for this package (default: no)
- `config`: _(struct `github.com/moby/tool/src/moby.ImageConfig`)_: Image configuration, marshalled to JSON and added as `org.mobyproject.config` label on image (default: no label)
- `depends`: Contains information on prerequisites which must be satisfied in order to build the package. Has subfields:
    - `docker-images`: Docker images to be made available (as `tar` files via `docker image save`) within the package build context. Contains the following nested fields:
        - `from-file` and `list`: _(string and string list respectively)_. Mutually exclusive fields specifying the list of images to include. Each image must include a valid digest (`sha256:...`) in order to maintain determinism. If `from-file` is used then it is a path relative to (and within) the package directory with one image per line (lines with `#` in column 0 and blank lines are ignore). If `list` is used then each entry is an image.
        - `target` and `target-dir`: _(string)_ Mutually exclusive fields specifying the target location, if `target` is used then it is a path relative to (and within) the package dir which names a `tar` file into which all of the listed images will be saved. If `target-dir` then it is a path relative to (and within) the package directory which names a directory into which each image will be saved (as `«image name»@«digest».tar`). **NB**: The path referenced by `target-dir` will be _removed_ prior to populating (to avoid issues with stale files).

## Building packages

### Prerequisites

Before you can build packages you need:
- Docker version 17.06 or newer. If you are on a Mac you also need
  `docker-credential-osxkeychain.bin`, which comes with Docker for Mac.
- `make`, `notary`, `base64`, `jq`, and `expect`
- A *recent* version of `manifest-tool` which you can build with `make
  bin/manifest-tool`, or `go get github.com:estesp/manifest-tool`, or
  via the LinuxKit homebrew tap with `brew install --HEAD
  manifest-tool`. `manifest-tool` must be in your path.
- The LinuxKit tool `linuxkit` which must be in your path.

Further, when building packages you need to be logged into hub with
`docker login` as some of the tooling extracts your hub credentials
during the build.

### Build packages as a maintainer

If you have write access to the `linuxkit` organisation on hub, you
should also be set up with signing keys for packages and your signing
key should have a passphrase, which we call `<passphrase>` throughout.

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
DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="<passphrase>" linuxkit pkg push --image=false --manifest «path-to-package»
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
DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE="<passphrase>" linuxkit pkg push «path-to-package»
```

#### Prerequisites

* For all of the steps, you *must* be logged into hub (`docker login`).
* For the manifest steps, you must be logged into hub and the passphrase for the key *must* be supplied as an environment variable. The build process has to resort to using `expect` to drive `notary` so none of the credentials can be entered interactively.

Since it is not very good to have your passphrase in the clear (or
even stashed in your shell history), we recommend using a password
manager with a CLI interface, such as LastPass or `pass`. You can then
invoke the build like this (for LastPass):

```
DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE=$(lpass show <key> --password) linuxkit pkg push «path-to-package»
```
or alternatively you may add the command to `~/.moby/linuxkit/config.yml` e.g.:
```
pkg:
  content-trust-passphrase-command: "lpass show <key> --password"
```

#### Signing Manually

If, for whatever reason, you want to sign an individual tag manually, whether the index (a.k.a. "multi-arch manifest") or the architecture-specific manifest, do the following:

1. Make sure you have ready your credentials:
   * docker hub login and passphrase
   * docker notary signing key passphrase
1. Get the following information:
   * the name of the image repository you want to sign, including the registry host but **not** including the tag, e.g. `linuxkit/containerd`
   * the tag of the image you want to sign, e.g. `a4aa19c608556f7d786852557c36136255220c1f` or `v5.0`
   * the size of the image you want to sign in bytes, e.g. `1052`. See below for information on how to get this.
   * the hash of the manifest or index to which the tag points, **not** including the `sha256:` leader, e.g. `66b3d74aeb855f393ddb85e7371a00d5f7994cc26b425825df2ce910583d74dc`. See below for information on how to get this.
1. Set env vars with the following:
   * `IMAGE`: name of the image, e.g. `IMAGE=docker.io/linuxkit/containerd`
   * `TAG`: the tag you want to sign. It could be a tag pointing at a multi-arch manifest or tag pointing at an individual architecture's manifest, e.g. `TAG=a4aa19c608556f7d786852557c36136255220c1f` or `TAG=a4aa19c608556f7d786852557c36136255220c1f-s390x`
   * `SIZE`: size of the pointed-at manifest or index, e.g. `SIZE=1052`
   * `HASH`: sha256 hash of the pointed-at manifest or index, e.g. `HASH=66b3d74aeb855f393ddb85e7371a00d5f7994cc26b425825df2ce910583d74dc`
1. Run the command: `notary -s https://notary.docker.io -d ~/.docker/trust addhash -p $IMAGE $TAG $SIZE --sha256 $HASH  -r targets/releases`

For example:

```console
IMAGE=docker.io/linuxkit/containerd
TAG=a4aa19c608556f7d786852557c36136255220c1f
SIZE=1052
HASH=66b3d74aeb855f393ddb85e7371a00d5f7994cc26b425825df2ce910583d74dc
notary -s https://notary.docker.io -d ~/.docker/trust addhash -p $IMAGE $TAG $SIZE --sha256 $HASH  -r targets/releases
```

##### Getting Size and Hash

There are several ways to get the size and hash of a particular manifest or index. Remember that you are signing a
tag, so you are looking for the size and hash of whatever the tag points to, manifest or index.

* `docker push`
* script
* `manifest-tool`
* `ocidist`

###### docker push

If you pushed the image tag using `docker push`, the very last line of output will give you the hash and size:

```console
$ docker push linuxkit/containerd:a4aa19c608556f7d786852557c36136255220c1f
The push refers to repository [docker.io/linuxkit/containerd]
fce5742422e4: Layer already exists
48a02e7b3096: Layer already exists
4381f8a59bb1: Layer already exists
c0328291406b: Layer already exists
79053b1996f5: Layer already exists
a4aa19c608556f7d786852557c36136255220c1f: digest: sha256:164f6c27410f145b479cdce1ed08e694c9b3d1e3e320c94d0e1ece9755043ea8 size: 1357
```

The first part is the tag you pushed, followed by the keyword `digest`, then the hash, then the size.

##### script

The following script command will provide the output for docker hub. Set the `IMAGE` name and `TAG`
environment variables.

```console
IMAGE=linuxkit/containerd
TAG=v0.8-amd64
jwt=$(curl -sSL "https://auth.docker.io/token?service=registry.docker.io&scope=repository:${IMAGE}:pull" | jq -r .token)
curl https://index.docker.io/v2/linuxkit/containerd/manifests/${TAG} -H "Authorization: Bearer ${jwt}" -H "Accept: application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json" -D /dev/stdout -o /dev/null -s
```

##### manifest-tool

The [manifest-tool](https://github.com/estesp/manifest-tool) allows you to inspect manifests, including
both OCI indexes, a.k.a. multi-arch manifests, and simple manifests.

If you inspect the actual tag, you will get just the hash, not the size.
If you inspect an index that includes a manifest that you want, you will get the hash and size.

For example, inspecting just a single arch manifest gives us the hash on the second line, but not the
size:

```console
$ manifest-tool inspect linuxkit/containerd:v0.8-amd64
Name: linuxkit/containerd:v0.8-amd64 (Type: application/vnd.docker.distribution.manifest.v2+json)
      Digest: sha256:0dc4f37966e23c0dffa6961119f29100c6d181b221e748c4688a280c08ab52a8
          OS: linux
        Arch: amd64
    # Layers: 5
      layer 1: digest = sha256:319073c03e01a960e61913b0e05b4e0094061726f6959732371a1496098c0980
      layer 2: digest = sha256:85521c11021aed78da3b61193b3e2cd1f316040eb08744f684cb98fa8ba35dc3
      layer 3: digest = sha256:f29bf65845868b4b2adccc661040b939e4119ca5b5cb34cb0583b8b4e279bcc9
      layer 4: digest = sha256:95c51328f79f6be125241ba10488e8962bdfd807fe93fc5d4d990eea7ac065e2
      layer 5: digest = sha256:794ca16dd5d22f1ccb5f58dea0ef9cb0c95d957ed33af5c4ab008cbdd30c359e
```

While inspecting the index that includes the above tag, gives us the hash but not the size of the
index, but finding the right entry, for example the first one is `amd64`, gives us the size as
`Mfst Length: 1357`:

```console
$ manifest-tool inspect linuxkit/containerd:v0.8
Name:   linuxkit/containerd:v0.8 (Type: application/vnd.docker.distribution.manifest.list.v2+json)
Digest: sha256:247e1eb712c2f5e9d80bb1a9ddf9bb5479b3f785a7e0dd4a8844732bbaa96851
 * Contains 3 manifest references:
1    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
1       Digest: sha256:0dc4f37966e23c0dffa6961119f29100c6d181b221e748c4688a280c08ab52a8
1  Mfst Length: 1357
1     Platform:
1           -      OS: linux
1           - OS Vers:
1           - OS Feat: []
1           -    Arch: amd64
1           - Variant:
1     # Layers: 5
         layer 1: digest = sha256:319073c03e01a960e61913b0e05b4e0094061726f6959732371a1496098c0980
         layer 2: digest = sha256:85521c11021aed78da3b61193b3e2cd1f316040eb08744f684cb98fa8ba35dc3
         layer 3: digest = sha256:f29bf65845868b4b2adccc661040b939e4119ca5b5cb34cb0583b8b4e279bcc9
         layer 4: digest = sha256:95c51328f79f6be125241ba10488e8962bdfd807fe93fc5d4d990eea7ac065e2
         layer 5: digest = sha256:794ca16dd5d22f1ccb5f58dea0ef9cb0c95d957ed33af5c4ab008cbdd30c359e

2    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
2       Digest: sha256:febd923be587826c64db19c429f92a369d6e41d8abb715ff30643250ceafa621
2  Mfst Length: 1357
2     Platform:
2           -      OS: linux
2           - OS Vers:
2           - OS Feat: []
2           -    Arch: arm64
2           - Variant:
2     # Layers: 5
         layer 1: digest = sha256:c35625c316366a48ec51192731e4155191b39fac7848e1b41fa46be1de9d11dc
         layer 2: digest = sha256:a73cb03ae4fe7b79bf9ec202ee734a55f962a597b93e9a9625c64e9f2be9e78f
         layer 3: digest = sha256:75b2023060fd85e40f4eed9fc5fe60c5b1866d909fc9ea783a21318ec2437e96
         layer 4: digest = sha256:413204d4c4ee875fd84dd93799ed1346cfb15e02a508b6306ea7da1a160babc3
         layer 5: digest = sha256:cf2293c110f0718e58e01ff4cbafa53eadde280999902fcdcd57269e8ba48339

3    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
3       Digest: sha256:b6adad183487d969059b3badeb5dce032bb449f61607eb024d91cfeabcaf0e57
3  Mfst Length: 1357
3     Platform:
3           -      OS: linux
3           - OS Vers:
3           - OS Feat: []
3           -    Arch: s390x
3           - Variant:
3     # Layers: 5
         layer 1: digest = sha256:16c1054185680ee839fa57dff29f412c179f1739191c12d33ab59bceca28a8ac
         layer 2: digest = sha256:e38fe65829ed75127337f18dc2a641e2e9f6c2859a314cf5ac1b7d5022150e26
         layer 3: digest = sha256:f2e84a29733f5f17cc860468b94eeeebf378d2a8af9bfc468427b1da430fe927
         layer 4: digest = sha256:b38f9350a90499ce01e7704a58b52c90ee28c5562379f7096ce930b5fea160be
         layer 5: digest = sha256:cc86a47d79015d074b41a4a3f0918e98dfb13f2fc6ef8def180a81fd36ae2544
```

##### ocidist

[ocidist](https://github.com/deitch/ocidist) is a simple utility to inspect or pull images, manifests,
indexes and individual blobs. If you call `ocidist manifest` and pass it the `--detail` flag, it will
report the hash and size.

For an index:

```console
$ ocidist manifest docker.io/linuxkit/containerd:v0.8 --detail
2020/11/12 11:00:03 ref name.Tag{Repository:name.Repository{Registry:name.Registry{insecure:false, registry:"index.docker.io"}, repository:"linuxkit/containerd"}, tag:"v0.8", original:"docker.io/linuxkit/containerd:v0.8"}
2020/11/12 11:00:03 advanced API
2020/11/12 11:00:06 referenced manifest hash sha256:247e1eb712c2f5e9d80bb1a9ddf9bb5479b3f785a7e0dd4a8844732bbaa96851 size 1052
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1357,
         "digest": "sha256:0dc4f37966e23c0dffa6961119f29100c6d181b221e748c4688a280c08ab52a8",
         "platform": {
            "architecture": "amd64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1357,
         "digest": "sha256:febd923be587826c64db19c429f92a369d6e41d8abb715ff30643250ceafa621",
         "platform": {
            "architecture": "arm64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1357,
         "digest": "sha256:b6adad183487d969059b3badeb5dce032bb449f61607eb024d91cfeabcaf0e57",
         "platform": {
            "architecture": "s390x",
            "os": "linux"
         }
      }
   ]
}
```

For a single manifest:

```console
$ ocidist manifest docker.io/linuxkit/containerd:v0.8-amd64 --detail
2020/11/12 10:59:08 ref name.Tag{Repository:name.Repository{Registry:name.Registry{insecure:false, registry:"index.docker.io"}, repository:"linuxkit/containerd"}, tag:"v0.8-amd64", original:"docker.io/linuxkit/containerd:v0.8-amd64"}
2020/11/12 10:59:08 advanced API
2020/11/12 10:59:11 referenced manifest hash sha256:0dc4f37966e23c0dffa6961119f29100c6d181b221e748c4688a280c08ab52a8 size 1357
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 1973,
      "digest": "sha256:b11103cf6c84fc3a2968d89e9d6fd7ce9e427380098c17828e3bda27de61ed6a"
   },
   "layers": [
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 41779632,
         "digest": "sha256:319073c03e01a960e61913b0e05b4e0094061726f6959732371a1496098c0980"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 328,
         "digest": "sha256:85521c11021aed78da3b61193b3e2cd1f316040eb08744f684cb98fa8ba35dc3"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 176,
         "digest": "sha256:f29bf65845868b4b2adccc661040b939e4119ca5b5cb34cb0583b8b4e279bcc9"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 202,
         "digest": "sha256:95c51328f79f6be125241ba10488e8962bdfd807fe93fc5d4d990eea7ac065e2"
      },
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 300,
         "digest": "sha256:794ca16dd5d22f1ccb5f58dea0ef9cb0c95d957ed33af5c4ab008cbdd30c359e"
      }
   ]
}
```



### Build packages as a developer

If you want to develop packages or test them locally, it is best to
override the hub organisation used. You may also want to disable
signing while developing. A typical example would be:

```
linuxkit pkg build -org=wombat -disable-content-trust «path-to-package»
```

This will create a local image: `wombat/<image>:<hash>-<arch>` which
you can use in your local YAML files for testing. If you need to test
on other systems you can push the image to your hub account and pull
from a different system by issuing:

```
linuxkit pkg build -org=wombat -disable-content-trust push
```

This will push both `wombat/<image>:<hash>-<arch>` and
`wombat/<image>:<hash>` to hub.

Finally, if you are tired of the long hashes you can override the hash
with:

```
linuxkit pkg build -org=wombat -disable-content-trust -hash=foo push
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
