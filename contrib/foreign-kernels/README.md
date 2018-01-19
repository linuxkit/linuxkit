# Using "foreign" kernels

This directory contains a number of scripts to re-package other
distributions kernels into a LinuxKit kernel package. The scripts
download the relevant `rpm`s or `deb`s and create a local docker image
which can be used in LinuxKit. You can optionally push the package to
hub, if you like.

All scripts take slightly different command line arguments (which
could be improved) as each distribution uses different naming
conventions and repository layouts.

## Example

To build a package using the `4.14.11` from the mainline [ppa
repository](http://kernel.ubuntu.com/~kernel-ppa/mainline), first
build the package:

```sh
./mainline.sh foobar/kernel-mainline v4.14.11 041411 201801022143
```

Here `v4.14.11` is the sub-directory of the [ppa
repository](http://kernel.ubuntu.com/~kernel-ppa/mainline), `041411`
seems to be another version used in the name of the `deb`s, and
`201801022143` is the date. You can find the names by browsing the
[ppa repository](http://kernel.ubuntu.com/~kernel-ppa/mainline).


The result is a local image `foobar/kernel-mainline:4.14.11`, which
can be used in a YAML file like a normal LinuxKit kernel image.
