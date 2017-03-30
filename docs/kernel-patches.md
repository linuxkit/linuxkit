# Working with Linux kernel patches for Moby

We may apply patches to the Linux kernel used in Moby, primarily to
cherry-pick some upstream patches or to add some additional
functionality, not yet accepted upstream.

Patches are located in `kernel/patches-<kernel version>` and should follow these rules:
- Patches *must* be in `git am` format, i.e. they should contain a
  complete and sensible commit message.
- Patches *must* contain a Developer's Certificate of Origin.
- Patch files *must* have a numeric prefix to ensure the ordering in
  which they are applied.
- If patches are cherry-picked, they *must* be cherry-picked with `-x`
  to contain the original commit ID.

This document outlines the recommended procedure to handle
patches. The general process is to apply them to a branch of the
[Linux stable tree](https://kernel.googlesource.com/pub/scm/linux/kernel/git/stable/linux-stable/)
and then export them with `git format-patch`.

If you want to add or remove patches currently used, please also ping
@rneugeba on the PR so that we can update our internal Linux tree to
ensure that patches are carried forward if we update the kernel in the
future.


# Preparation

Patches are applied to point releases of the linux stable tree. You need an up-to-date copy of that tree:
```sh
git clone git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git
```

We use the following variables:
- `MOBYSRC`: Base directory of Moby Linux repository
- `LINUXSRC`: Base directory of Linux stable kernel repository
e.g.:
```sh
MOBYSRC=~/src/docker/moby
LINUXSRC=~/src/docker/linux-stable
```
to refer to the location of the Moby and Linux kernel trees.


# Updating the patches to a new kernel version

There are different ways to do this, but we recommend applying the patches to the current version and then rebase to the new version. We define the following variables to refer to the current base tag and the new tag you want to rebase the patches to:
```sh
CURTAG=v4.9.14
NEWTAG=v4.9.15
```

If you don't already have a branch, it's best to import the current patch set and then rebase:
```sh
cd $LINUXSRC
git checkout -b ${NEWTAG}-moby ${CURTAG}
git am ${MOBYSRC}/kernel/patches/*.patch
git rebase ${NEWTAG}-moby ${NEWTAG}
```

The `git am` should not have any conflicts and if the rebase has conflicts resolve them, then `git add <files>` and `git rebase --continue`.

If you already have linux tree with a `${CURTAG}-moby` branch, you can rebase by creating a new branch from the current branch and then rebase:
```sh
cd $LINUXSRC
git checkout ${CURTAG}-moby
git branch ${NEWTAG}-moby ${CURTAG}-moby
git rebase --onto ${NEWTAG} ${NEWTAG} ${NEWTAG}-moby
```
Again, resolve any conflicts as described above.


# Adding/Removing patches

If you want to add or remove patches make sure you have an up-to-date branch with the currently applied patches (see above). Then either any normal means (`git cherry-pick -x`, `git am`, or `git commit`, etc) to add new patches. For cherry-picked patches also please add a `Origin:` line after the DCO lines with a reference the git tree the patch was cherry-picked from.

If the patch is not cherry-picked try to include as much information
in the commit message as possible as to where the patch originated
from. The canonical form would be to add a `Origin:` line after the
DCO lines, e.g.:
```
Origin: https://patchwork.ozlabs.org/patch/622404/
```

# Export patches to moby

To export patches to Moby, you should use `git format-patch` from the Linux tree, e.g., something along these lines:
```sh
cd $LINUXSRC
rm $MOBYSRC/kernel/patches-4.9/*
git format-patch -o $MOBYSRC/kernel/patches-4.9 v4.9.15..HEAD
```

The, create a PR for Moby.
