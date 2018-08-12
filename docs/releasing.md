# Making a LinuxKit release

This document describes the steps to make a LinuxKit release. A
LinuxKit release consists of:
- A git tag of the form vX.Y on a specific commit.
- Packages on Docker hub, tagged with the release tag.
- All sample `YAML` files updated to use the release packages
- `linuxkit` binaries for all supported architectures.
- Changelog entry

Note, we explicitly do not tag kernel images with LinuxKit release
tags as we encourage users to stay current with the kernel
releases. We also do not tag test and `mkimage` packages as these are
not end-user facing.


## Pre-requisites

Releases can be done by any maintainer. Maintainers need to have
access to build machines for all architectures support by LinuxKit and
signing keys set up to sign Docker hub images.


## Release preparation

The release preparation is by far the most time consuming task as it
involves updating all packages and YAML files.

The release preparation is performed on a branch of your up-to-date
LinuxKit clone. This document assumes that your clone of the LinuxKit
repository is available as the `origin` remote in your local `git`
clone (in my setup the official LinuxKit repository is available as
`upstream` remote). If your setup is different, you may have to adjust
some of the commands below.

As a starting point you have to be on the update to date master branch
and be in the root directory of your local git clone. You should also
have the same setup on all build machines used.

To make the release steps below cut-and-pastable, define the following
environment variables:

```sh
LK_RELEASE=v0.4
LK_ROOT=$(pwd)
LK_REMOTE=origin
```

On one of the build machines (preferably the `x86_64` machine), create
the release branch:

```sh
git checkout -b rel_$LK_RELEASE
```

Also make sure that you have a recent version of the `linuxkit`
utility in the path. Either a previous release or compiled from
master.


### Update `linuxkit/alpine`

This step is not necessarily required if the alpine base image has
recently been updated, but it is good to pick up any recent bug
fixes. Updating the alpine base image is different to other packages
and it must be performed on `x86_64` first:

```sh
cd $LK_ROOT/tools/alpine
make push
```

This will update `linuxkit/alpine` and change the `versions.x86_64`
file. Check it in and push to GitHub:

```sh
git commit -a -s -m "tools/alpine: Update to latest"
git push $LK_REMOTE rel_$LK_RELEASE
```

Now, on each build machine for the other supported architectures, in turn:

```sh
git fetch
git checkout rel_$LK_RELEASE
cd $LK_ROOT/tools/alpine
make push
git commit -a --amend
git push --force $LK_REMOTE rel_$LK_RELEASE
```

With all supported architectures updated, head back to the `x86_64`
machine and update the release branch:

```sh
git fetch && git reset --hard $LK_REMOTE/rel_$LK_RELEASE
```

Stash the tag of the alpine base image in an environment variable:

```sh
LK_ALPINE=$(head -1 alpine/versions.x86_64 | sed 's,[#| ]*,,' | sed 's,\-.*$,,' | cut -d':' -f2)
```


### Update tools packages

On the `x86_64` machine, get the `linuxkit/alpine` tag and update the
other packages:

```sh
cd $LK_ROOT/tools
../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE
git checkout alpine/versions.aarch64 alpine/versions.s390x

git commit -a -s -m "tools: Update to the latest linuxkit/alpine"
git push $LK_REMOTE rel_$LK_RELEASE

make forcepush
```

Note, the `git checkout` reverts the changes made by
`update-component-sha.sh` to files which are accidentally updated and
the `make forcepush` will skip building the alpine base.

Then, on the other build machines in turn:

```sh
cd $LK_ROOT/tools
git fetch && git reset --hard $LK_REMOTE/rel_$LK_RELEASE
make forcepush
```

Back on the `x86_64` machine:

```sh
cd $LK_ROOT
for img in $(cd tools; make show-tag); do
    ./scripts/update-component-sha.sh --image $img
done

git commit -a -s -m "Update use of tools to latest"
```


### Update test packages

Next, we update the test packages to the updated alpine base on the `x86_64` system:

```sh
cd $LK_ROOT/test/pkg
../../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "tests: Update packages to the latest linuxkit/alpine"
git push $LK_REMOTE rel_$LK_RELEASE

make push
```

Then, on the other build machines in turn:

```sh
cd $LK_ROOT/test/pkg
git fetch && git reset --hard $LK_REMOTE/rel_$LK_RELEASE
make push
```

Back on the `x86_64` machine:

```sh
cd $LK_ROOT
for img in $(cd test/pkg; make show-tag); do
    ./scripts/update-component-sha.sh --image $img
done

git commit -a -s -m "Update use of test packages to latest"
```

Some tests also use `linuxkit/alpine`. Update them as well:

```sh
cd $LK_ROOT/test/cases
../../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "tests: Update tests cases to the latest linuxkit/alpine"
```

### Update packages

Next, we update the LinuxKit packages. This is really the core of the
release. The other steps above are just there to ensure consistency
across packages.


```sh
cd $LK_ROOT/pkg
../scripts/update-component-sha.sh --image linuxkit/alpine:$LK_ALPINE

git commit -a -s -m "pkgs: Update packages to the latest linuxkit/alpine"
git push $LK_REMOTE rel_$LK_RELEASE
```

Most of the packages are build from `linuxkit/alpine` and source code
in the `linuxkit` repository, but some packages wrap external
tools. The time of a release is a good opportunity to check if there
have been updates. Specifically:

- `pkg/cadvisor`: Check for [new releases](https://github.com/google/cadvisor/releases).
- `pkg/firmware` and `pkg/firmware-all`: Use latest commit from [here](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git).
- `pkg/node_exporter`: Check for [new releases](https://github.com/prometheus/node_exporter/releases).
- Check [docker hub](https://hub.docker.com/r/library/docker/tags/) for the latest `dind` tags. and update `examples/docker.yml`, `examples/docker-for-mac.yml`, `examples/cadvisor.yml`, and `test/cases/030_security/000_docker-bench/test.yml` if necessary.

The build/push the packages:

```sh
cd $LK_ROOT/pkg
make OPTIONS="-release $LK_RELEASE" push
```

Note, the `OPTIONS` argument. This adds the release tag to the
packages.

Then, on the other build machines in turn:

```sh
cd $LK_ROOT/pkg
git fetch && git reset --hard $LK_REMOTE/rel_$LK_RELEASE
make OPTIONS="-release $LK_RELEASE" push
```

Update the package tags in the YAML files:

```sh
cd $LK_ROOT
for img in $(cd pkg; make show-tag | cut -d ':' -f1); do
    ./scripts/update-component-sha.sh --image $img:$LK_RELEASE
done

git commit -a -s -m "Update package tags to $LK_RELEASE"
```

### Final preparation steps

- Update AUTHORS by running `./scripts/generate-authors.sh`
- Update the `VERSION` variable in the top-level `Makefile`
- Create an entry in `CHANGELOG.md`. Take a look at `git log v0.3..HEAD` and pick interesting updates (of course adjust `v0.3` to the previous version).
- Create a PR with your changes.


## Releasing

Once the PR is merged we can do the actual release.

- Update your local git clone to the lastest
- Identify the merge commit for your PR and tag it and push it to the main LinuxKit repository (remote `upstream` in my case):

```
git tag $LK_RELEASE master
git push upstream $LK_RELEASE
```

Then head over to GitHub and look at the `Releases` tab. You should see the new tag. Edit it:
- Add the changelog message
- Head over to the Circle CI page of the master build (try the Circle CI badge in the top level `README.md`)
- Download the artefacts and SHA256 sums file.
- Add the downloaded binaries to the release page (drag-and-drop below the editor window)
- Add the `sha256` sums to the release notes on the release page

Hit the `Publish release` button.

This completes the release, but you are not done, one more step is required.

## Post release

Create a PR which bumps the version number in the top-level `Makefile`
to `$LK_RELEASE+` to make sure that the version reported by `linuxkit
version` gets updated.


