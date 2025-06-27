# Image Caching

linuxkit builds each runtime OS image from a combination of Docker images.
These images are pulled from a registry and cached locally.

linuxkit does not use the docker image cache to store these images. This is
for two key reasons.

First, docker does not provide support for different architecture versions. For
example, if you want to pull down `docker.io/library/alpine:3.13` by manifest,
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
        "org.opencontainers.image.ref.name": "docker.io/library/alpine:3.13"
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

## Imports from local Docker instance

To import an image from your local Docker daemon into LinuxKit, you’ll need to ensure the image is exported in the [OCI image format](https://docs.docker.com/build/exporters/oci-docker/), which LinuxKit understands.

This requires using a `docker-container` [buildx driver](https://docs.docker.com/build/builders/drivers/docker-container/), rather than the default.

Set it up like so:

```shell
docker buildx create --driver docker-container --driver-opt image=moby/buildkit:latest --name=ocibuilder --bootstrap
```

Then build and export your image using the OCI format:

```shell
docker buildx build --builder=ocibuilder --output type=oci,name=foo . > foo.tar
```

You can now import it into LinuxKit with:

```shell
linuxkit cache import foo.tar
```

Note that this process, as described, will only produce images for the platform/architecture you're currently on. To produce multi-platform images requires extra docker build flags and external builder or QEMU support - see [here](https://docs.docker.com/build/building/multi-platform/).

This workaround is only necessary when working with the local Docker daemon. If you’re pulling from Docker Hub or another registry, you don’t need to do any of this.
