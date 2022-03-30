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

### Update `linuxkit/alpine`

This step is not necessarily required if the alpine base image has
recently been updated, but it is good to pick up any recent bug
fixes. Follow the process in [alpine-base-update.md](./alpine-base-update.md)

There are several important notes to consider when updating alpine base:

* `LK_BRANCH` is set to `rel_$LK_RELEASE`, when cutting a release, for e.g. `LK_BRANCH=rel_v0.9`
* It not necessarily required to update the alpine base image if it has recently been updated, but it is good to pick up any recent bug
fixes. However, you do need to update the tools, packages and tests.
* Releases are a particularly good time to check for updates in wrapped external dependencies, as highlighted in [alpine-base-update.md#External Tools](./alpine-base-update.md#External_Tools)

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
