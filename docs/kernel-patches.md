# Working with Linux kernel patches for Moby

We may apply patches to the Linux kernel used in Moby, primarily to
cherry-pick some upstream patches or to add some additional
functionality, not yet accepted upstream.  This document outlines the
recommended procedure to handle these patches.

Patches are located in `alpine/kernel/patches` and are maintained in
`git am` format to keep important meta data such as the provenance of
the patch.


# Preparation

Patches are applied to point releases of the linux stable tree. You need an up-to-date copy of that tree:
```sh
git clone git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git
```

Throughout we use the following variables:
- `MOBYSRC`: Base directory of Moby Linux repository
- `LINUXSRC`: Base directory of Linux stable kernel repository
- `LINUXTAG`: Release tag to base the patches on
e.g.:
```sh
MOBYSRC=~/src/docker/moby
LINUXSRC=~/src/docker/linux-stable
LINUXTAG=v4.4.24
```

# Updating the patches to a new kernel version

Create a branch from a tag for the new patches, e.g.:
```sh
cd $LINUXSRC
git branch ${LINUXTAG}-moby ${LINUXTAG}
git checkout ${LINUXTAG}-moby
```

Import all the existing patches:
```sh
cd $LINUXSRC
git am ${MOBYSRC}/alpine/kernel/patches/*.patch
```

If this causes merge conflicts resolve them as they arise and continue as instructed.

Once finished, update the patches in Moby:
```sh
cd $LINUXSRC
rm $MOBYSRC/alpine/kernel/patches
git format-patch -o $MOBYSRC/alpine/kernel/patches ${LINUXTAG}..HEAD
```

Create a PR for Moby.


# Adding new patches

For patches from upstream Linux kernel versions, use cherry-picking:
```sh
git cherry-pick -x <sha of commit>
```
The `-x` ensures that the origin of the patch is recorded in the commit.


For patches from the mailing list or patchworks, add a line like:
```
Origin: https://patchwork.ozlabs.org/patch/622404/
```
to the patch (after the `Signed-off-by` and `Cc` lines.


For patches written from scratch, make sure it has a sensible commit
messages as well as a DCO line.
