# Signing LinuxKit Hub Images

We sign and verify LinuxKit component images, such as `linuxkit/kernel`, using [Notary](https://github.com/docker/notary).

This document details the process for setting this up, intended for maintainers.

## Initialize a New Repository

Let's say we're publishing a new `linuxkit/foo` image that we want to sign and verify in LinuxKit.
We first need to initialize the Notary repository:

```
notary -s https://notary.docker.io -d ~/.docker/trust init -p docker.io/linuxkit/foo
```

This command will generate some private keys in `~/.docker/trust` and ask you for passphrases such that they are encrypted at rest.
All linuxkit repositories are currently using the same root key so we can pin trust on key ID `1908a0cf4f55710138e63f65ab2a97e8fa3948e5ca3b8857a29f235a3b61ea1b`.

We'll also let the notary server take control of the snapshot key, for easier delegation collaboration:
```
notary -s https://notary.docker.io -d ~/.docker/trust key rotate docker.io/linuxkit/foo snapshot -r
```

## Add maintainers to delegation roles:

Maintainers are to sign with `delegation` keys, which are adminstered by a non-root key.
Thusly, they are easily rotated without having to bring the root key online.
Additionally, maintainers can be added to separate roles for auditing purposes: the current setup is to add maintainers to both the `targets/releases` role that is intended
for release consumption, as well as an individual `targets/<maintainer_name>` role for auditing.
Docker will automatically sign into both roles when pushing with Docker Content Trust.

Here's what the command looks like to add all maintainers to the `targets/releases` role:
```
notary -s https://notary.docker.io -d ~/.docker/trust delegation add -p docker.io/linuxkit/foo targets/releases alice.crt bob.crt charlie.crt --all-paths
```

Here's what the commands look like to add all maintainers to their individually named roles:
```
notary -s https://notary.docker.io -d ~/.docker/trust delegation add -p docker.io/linuxkit/foo targets/alice alice.crt --all-paths
notary -s https://notary.docker.io -d ~/.docker/trust delegation add -p docker.io/linuxkit/foo targets/bob bob.crt --all-paths
notary -s https://notary.docker.io -d ~/.docker/trust delegation add -p docker.io/linuxkit/foo targets/charlie charlie.crt --all-paths
```

## Maintainers import their private keys

It's important that each maintainer imports their private key into Docker's key storage, so Docker can use it to sign:
```
notary -d ~/.docker/trust key import alice.key -r user
```
