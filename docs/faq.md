# FAQ

Please open an issue if you want to add a question here.

## How do updates work?

LinuxKit does not require being installed on a disk, it is often run from an ISO, PXE or other
such means, so it does not require an on disk upgrade method such as the ChromeOS code that
is often used. It would definitely be possible to use that type of upgrade method if the 
system is installed, and it would be useful to support this for that use case, and an
updater container to control this for people who want to use this.

We generally use external tooling such as [Infrakit](https://github.com/docker/infrakit) or
CloudFormation templates to manage the update process externally from LinuxKit, including
doing rolling cluster upgrades to make sure distributed applications stay up and responsive.

Updates may preserve the state disk used by applications if needed, either on the same physical
node, or by reattaching a virtual cloud volume to a new node.

## What do I need to build LinuxKit?

We have tried to make this as simple as possible, by using containers for the build process, so
you should be able to build LinuxKit on any OSX or Linux laptop; we should have Windows build support
soon.

## Why not use `systemd`?

In order to keep the system minimal, `systemd` did not seem appropriate, as it brings in a lot
of dependencies and functionality that we do not need. At present we are using the `busybox`
`init` process, and a small set of minimal scripts, but we expect to replace that with a small
standalone `init` process and a small piece of code to bring up the system containers where the
real work takes place.
