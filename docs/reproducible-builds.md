# Reproducible builds

We aim to make the outputs of `linuxkit build` reproducible, i.e. the
build artefacts should be bit-by-bit identical copies if invoked with
the same inputs and run with the same version of the `linuxkit`
command. See [this
document](https://reproducible-builds.org/docs/buy-in/) on why this
matters.

_Note, we do not (yet) aim to make `linuxkit pkg build` builds
reproducible._


## Current status

Currently, the following output formats provide reproducible builds:
- `tar` (Tested as part of the CI)
- `tar-kernel-initrd`
- `docker`
- `kernel+initrd` (Tested as part of the CI)


## Details

In general, `linuxkit build` lends itself for reproducible
builds. LinuxKit packages, used during `linuxkit build`, are (signed)
docker images. Packages are tagged with the content hash of the source
code (and optionally release version) and are typically only updated
if the source of the package changed (in which case the tag
changes). For all intents and purposes, when pulled by tag, the
contents of a packages should be bit-by-bit identical. Alternatively,
the digest of the package, in which case, the pulled image will always
be the same.

The first phase of the `linuxkit build` mostly untars and retars the
images of the packages to produce an tar file of the root filesystem.
This then serves as input for other output formats. During this first
phase, there are a number of things to watch out for to generate
reproducible builds:

- Timestamps of generated files. The `docker export` command, as well
  as `linuxkit build` itself, creates a small number of files. The
  `ModTime` for these files needs to be clamped to a fixed date
  (otherwise the current time is used). Use the `defaultModTime`
  variable to set the `ModTime` of created files to a specific time.
- Generated JSON files. `linuxkit build` generates a number of JSON
  files by marshalling Go `struct` variables. Examples are the OCI
  specification `config.json` and `runtime.json` files for
  containers. The default Go `json.Marshal()` function seems to do a
  reasonable good job in generating reproducible output from internal
  structures, including for JSON objects. However, during `linuxkit
  build` some of the OCI runtime spec fields are generated/modified
  and care must be taken to ensure consistent ordering. For JSON
  arrays (Go slices) it is best to sort them before Marshalling them.

Reproducible builds for the first phase of `linuxkit build` can be
tested using `-output tar` and comparing the output of subsequent
builds with tools like `diff` or the excellent
[`diffoscope`](https://diffoscope.org/).

The second phase of `linuxkit build` converts the intermediary `tar`
format into the desired output format. Making this phase reproducible
depends on the tools used to generate the output.

Builds, which produce ISO formats should probably be converted to use
[`go-diskfs`](https://github.com/diskfs/go-diskfs) before attempting
to make them reproducible.

For ideas on how to make the builds for other output formats
reproducible, see [this
page](https://reproducible-builds.org/docs/system-images/).
