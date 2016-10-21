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
- `CURTAG`: Release tag patches are currently based on
- `NEWTAG`: New release tag to base the patches on
e.g.:
```sh
MOBYSRC=~/src/docker/moby
LINUXSRC=~/src/docker/linux-stable

CURTAG=v4.4.23
NEWTAG=v4.4.24
```


# Updating the patches to a new kernel version

There are different ways to do this. You can either rebase or try to
re-apply the patches.  rebase is the recommended way. Once you have
the patches in a new branch you need to export them.

## Rebase

The simplest way is to create a new branch of the current tag, apply the patches and then rebase to the new tag:
```sh
cd $LINUXSRC
git checkout -b ${NEWTAG}-moby ${CURTAG}
git am ${MOBYSRC}/alpine/kernel/patches/*.patch
git rebase ${NEWTAG}-moby ${NEWTAG}
```

The `git am` should not have any conflicts and if the rebase has conflicts resolve them, then `git add <files>` and `git rebase --continue`.

If you already have a `${CURTAG}-moby` branch, you can also do a more complex rebase by creating a new branch from the current branch and then rebase:
```sh
cd $LINUXSRC
git checkout ${CURTAG}-moby
git branch ${NEWTAG}-moby ${CURTAG}-moby
git rebase --onto ${NEWTAG} ${NEWTAG} ${NEWTAG}-moby
```
Again, resolve any conflicts as described above.


## Re-apply patches

Create a branch from a tag for the new patches, e.g.:
```sh
cd $LINUXSRC
git checkout -b ${NEWTAG}-moby ${NEWTAG}
```

Import all the existing patches into the new branch:
```sh
cd $LINUXSRC
git am --reject ${MOBYSRC}/alpine/kernel/patches/*.patch
```

If this causes merge conflicts resolve them as they arise and continue as instructed.


## Export patches to moby

Irrespective of using the rebase or re-apply method, you should now have a `${NEWTAG}-moby` branch. Form this export the patches to moby:
```sh
cd $LINUXSRC
rm $MOBYSRC/alpine/kernel/patches/*
git format-patch -o $MOBYSRC/alpine/kernel/patches ${NEWTAG}..HEAD
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
