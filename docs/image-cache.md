# Image Caching

linuxkit builds each runtime OS image from a combination of Docker images.
These images are pulled from a registry and cached locally.

linuxkit does not use the docker image cache to store these images. This is
for two key reasons.

First, docker does not provide support for different architecture versions. For
example, if you want to pull down `docker.io/library/alpine:3.11` by manifest,
with its signature, but get the `arm64` version while you are on an `amd64` device,
it is not supported.

Second, and more importantly, this requires a running docker daemon. Since the
very essence of linuxkit is removing daemons and operating systems where unnecessary,
just laying down bits in a file, removing docker from the image build process
is valuable. It also simplifies many use cases, like CI, where a docker daemon
may be unavailable.

## How LinuxKit Caches Images

LinuxKit pulls images down from a registry and stores them in a local cache.
It stores the root manifest or index of the image, the manifest, and all of the layers
for the requested architecture. It does not pull down layers, manifest or config
for all available architectures, only the requested one. If none is requested, it
defaults to the architecture on which you are running.

By default, LinuxKit caches images in `~/.linuxkit/cache/`. It can be changed
via a command-line option. The structure of the cache directory matches the
[OCI spec for image layout](http://github.com/opencontainers/image-spec/blob/master/image-layout.md).

Image names are kept in `index.json` in the [annotation](https://github.com/opencontainers/image-spec/blob/master/annotations.md) `org.opencontainers.image.ref.name`. For example"

```json
{
  "schemaVersion": 2,
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
      "size": 1638,
      "digest": "sha256:9a839e63dad54c3a6d1834e29692c8492d93f90c59c978c1ed79109ea4fb9a54",
      "annotations": {
        "org.opencontainers.image.ref.name": "docker.io/library/alpine:3.11"
      }
    }
  ]
}
```

## How LinuxKit Uses the Cache and Registry

For each image that linuxkit needs to read, it does the following. Note that if the `--pull` option
is provided, it always will pull, independent of what is in the cache.

1. Check in the cache for the image name in the cache `index.json`. If it does not find it, pull it down and store it in cache.
1. Read the root hash from `index.json`.
1. Find the root blob in the `blobs/` directory via the hash and read it.
1. Proceed to read the manifest, config and layers.

The read process is smart enough to check each blob in the local cache before downloading
it from a registry.
