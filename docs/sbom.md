# Software Bill-of-Materials

LinuxKit bootable images are composed of existing OCI images.
OCI images, when built, often are scanned to create a
software bill-of-materials (SBoM). The buildkit builder
system itself contains the [ability to integrate SBoM scanning and generation into the build process](https://docs.docker.com/build/attestations/sbom/).

When LinuxKit composes an operating system image using `linuxkit build`,
it will, by default, combine the SBoMs of all the OCI images used to create
the final image.

It looks for SBoMs in the following locations:

* [image attestation storage](https://docs.docker.com/build/attestations/attestation-storage/)

Future support for [OCI Image-Spec v1.1 Artifacts](https://github.com/opencontainers/image-spec)
is under consideration, and will be reviewed when it is generally available.

When building packages with `linuxkit pkg build`, it also has the ability to generate an SBoM for the
package, which later can be consumed by `linuxkit build`.

## Consuming SBoM From Packages

When `linuxkit build` is run, it does the following for dealing with SBoMs:

1. For each OCI image that it processes:
   1. check if the image contains an SBoM attestation; it not, skip this step.
   1. Retrieve the SBoM attestation.
1. After generating the root filesystem, combine all of the individual SBoMs into a single unified SBoM.
1. Save the output single SBoM into the root of the image as `sbom.spdx.json`.

Currently, only SPDX json format is supported.

### SBoM Scanner and Output Format

By default, linuxkit combines the SBoMs into a file with output format SPDX json,
and the file saved to the filename `sbom.spdx.json`.

In addition, in order to assist with reproducible builds, the creation date/time of the SBoM is
a fixed date/time set by linuxkit, rather than the current date/time. Note, however, that even
with a fixed date/time, reproducible builds depends on reproducible SBoMs on the underlying container images.
This is not always the case, as the unique IDs for each package and file might be deterministic, but it might not.

This can be overridden by using the CLI flags:

* `--no-sbom`: do not find and consolidate the SBoMs
* `--sbom-output <filename>`: the filename to save the output to in the image.
* `--sbom-current-time true|false`: whether or not to use the current time for the SBoM creation date/time (default `false`)

### Disable SBoM for Images

To disable SBoM generation when running `linuxkit build`, use the CLI flag `--sbom false`.

## Generating SBoM For Packages

When `linuxkit pkg build` is run, by default it enables generating an SBoM using the
[SBoM generating capabilities of buildkit](https://www.docker.com/blog/generate-sboms-with-buildkit/).
This means that it inherits all of those capabilities as well, and saves the SBoM in the same location,
as an attestation on the image.

### SBoM Scanner

By default, buildkit runs [syft](http://hub.docker.com/r/anchore/syft) with output format SPDX json,
specifically via its integration image [buildkit-syft-scanner](docker.io/docker/buildkit-syft-scanner).
You can select a different image to run a scanner, provided it complies with the
[buildkit SBoM protocol](https://github.com/moby/buildkit/blob/master/docs/attestations/sbom-protocol.md),
by passing the CLI flag `--sbom-scanner <image>`.

### Disable SBoM for Packages

To disable SBoM generation when running `linuxkit pkg build`, use the CLI flag `--sbom-scanner=false`.

