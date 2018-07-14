## Private Images
When building, `moby` downloads, and optionally checks the notary signature, on any OCI images referenced in any section. 

As of this writing, `moby` does **not** have the ability to download these images from registries that require credentials to access. This is equally true for private images on public registries, like https://hub.docker.com, as for private registries.

We are working on enabling private images with credentials. Until such time as that feature is added, you can follow these steps to build a moby image using OCI images
that require credentials to access:

1. `docker login` as relevant to authenticate against the desired registry.
2. `docker pull` to download the images to your local machine where you will run `moby build`.
3. Run `moby build` (or `linuxkit build`).

Additionally, ensure that you do **not** have trust enabled for those images. See the section on [trust](#trust) in this document. Alternately, you can run `moby build` or `linuxkit build` with `--disable-trust`.
