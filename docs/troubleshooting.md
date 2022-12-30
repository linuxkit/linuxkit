# Troubleshooting

This document contains a list of known issues related to using, building or testing linuxkit.

## Images

## Packages

### Invalid MediaType

**Problem**

```
Error: error building and pushing "linuxkit/mkimage-iso-efi-initrd:0e66171ffde9bb735b0e014f811f9626fc8b9bc9": PUT https://index.docker.io/v2/linuxkit/mkimage-iso-efi-initrd/manifests/0e66171ffde9bb735b0e014f811f9626fc8b9bc9: MANIFEST_INVALID: manifest invalid; if present, mediaType in image index should be 'application/vnd.oci.image.index.v1+json' not 'application/vnd.docker.distribution.manifest.list.v2+json'
```

The above message is caused by registries, notably docker hub, refusing to accept indexes with the
docker media type of `application/vnd.docker.distribution.manifest.list.v2+json`, rather than the OCI
one `application/vnd.oci.image.index.v1+json`.

Linuxkit _does_ use the OCI media type, however, if the image _already_ exists in the registry, linuxkit will
pull the index down, update it, and push it back up. The above error occurs because the index that exists in
the hub, the one that is pulled down, has the older media type, from when the registry accepted it.

**Solution**

The solution is to force an entirely new build, which will generate the images and index with the correct media
type.

```
linuxkit pkg build --force <path>
linuxkit pkg push <path>
```

## Testing

