# Yaml configuration

The yaml configuration specifies the components and the build time artifacts. All components
are downloaded at build time to create an image. The image is self-contained and immutable,
so it can be tested reliably for continuous delivery.

The configuration file is processed in the order `kernel`, `init`, `onboot`, `services`, `files`.
Each section adds file to the root file system

## `kernel`

This section defines the kernel configuration. The `image` field specifies the Docker image,
which should contain a `bzImage` (for `amd64` architecture, others may vary) and a file
called `kernel.tar` which is a tarball that is unpacked into the root, which should usually
contain a kernel modules directory. See [`kernel/`](../kernel/) for source code. `cmdline`
specifies the kernel command line options if required.

## `init`

This section currently just lists images that is used for the `init` system and are unpacked directly
into the root filesystem. This should bring up `containerd`, start the system and daemon containers,
and set up basic filesystem mounts. See [`pkg/init/`](../pkg/init/) for source code. For ease of
modification `runc` and `containerd` images, which just contain these programs are added here
rather than bundled into the `init` container.

## `onboot`

These containers are run to completion sequentially, using `runc` before anything else is started.
They can be used to configure one shot settings. For details of the config for each container, see
below.

## `services`

These containers are started with `containerd` and are expected to remain running. Startup order
is not guaranteed, so containers should wait on any resources, such as networking, that they need.
For details of the config for each container, see below.

## `trust`

This section specifies which build components are to be cryptographically verified with
[Docker Content Trust](https://docs.docker.com/engine/security/trust/content_trust/) prior to pulling.
Trust is a central concern in any build system, and LinuxKit's is no exception: Docker Content Trust provides authenticity,
integrity, and freshness guarantees for the components it verifies.  The LinuxKit maintainers are responsible for signing
`mobylinux` components, though collaborators can sign their own images with Docker Content Trust or [Notary](https://github.com/docker/notary).

- `image` lists which individual images to enforce pulling with Docker Content Trust.
The image name may include tag or digest, but the matching also succeeds if the base image name is the same.
- `org` lists which organizations for which Docker Content Trust is to be enforced across all images (ex: `mobylinux` is the org for `mobylinux/kernel`)

## Image specification

For each image in the `system` and `daemon` sections you can specify the OCI options that are passed to
`runc`, so you can specify what capabilities are needed and so on. Generally there are few defaults.
For more details see the [OCI specification](https://github.com/opencontainers/runtime-spec/blob/master/spec.md).

- `name` a unique name for the program being executed, used as the `containerd` id.
- `image` the Docker image to use for the root filesystem. The default command, path and environment are
  extracted from this so they need not be filled in.
- `capabilities` the Linux capabilities required, for example `CAP_SYS_ADMIN`. If there is a single
  capability `all` then all capabilities are added.
- `mounts` is the full form for specifying a mount, which requires `type`, `source`, `destination`
  and a list of `options`. If any fields are omitted, sensible defaults are used if possible, for example
  if the `type` is `dev` it is assumed you want to mount at `/dev`. The default mounts and their options
  can be replaced by specifying a mount with new options here at the same mount point.
- `binds` is a simpler interface to specify bind mounts, accepting a string like `/src:/dest:opt1,opt2`
  similar to the `-v` option for bind mounts in Docker.
- `tmpfs` is a simpler interface to mount a `tmpfs`, like `--tmpfs` in Docker, taking `/dest:opt1,opt2`.
- `command` will override the command and entrypoint in the image with a new list of commands.
- `env` will override the environment in the image with a new environment list
- `cwd` will set the working directory, defaults to `/`.
- `net` sets the network namespace, either to a path, or if `host` is specified it will use the host namespace.
- `pid` sets the pid namespace, either to a path, or if `host` is specified it will use the host namespace.
- `ipc` sets the ipc namespace, either to a path, or if `host` is specified it will use the host namespace.
- `uts` sets the uts namespace, either to a path, or if `host` is specified it will use the host namespace.
- `readonly` sets the root filesystem to read only, and changes the other default filesystems to read only.
- `maskedPaths` sets paths which should be hidden.
- `readonlyPaths` sets paths to read only.
- `uid` sets the user id of the process. Only numbers are accepted.
- `gid` sets the group id of the process. Only numbers are accepted.
- `additionalGids` sets additional groups for the process. A list of numbers is accepted.
- `noNewPrivileges` is `true` means no additional capabilities can be acquired and `suid` binaries do not work.
- `hostname` sets the hostname inside the image.
- `oomScoreAdj` changes the OOM score.
- `disableOOMKiller` disables the OOM killer for the service.
- `rootfsPropagation` sets the rootfs propagation, eg `shared`, `slave` or (default) `private`.
- `cgroupsPath` sets the path for cgroups.
- `sysctl` sets a list of `sysctl` key value pairs that are set inside the container namespace.

Further OCI values will be added, as the list is not yet complete.

### Mount Options
When mounting filesystem paths into a container - whether as part of `onboot` or `services` - there are several options of which you need to be aware. Using them properly is necessary for your containers to function properly.

For most containers - e.g. nginx or even docker - these options are not needed. Simply doing the following will work fine:

```yml
binds:
 - /var:/some/var/path
```

However, in some circumstances you will need additional options. These options are used primarily if you intend to make changes to mount points _from within your container_ that should be visible from outside the container, e.g., if you intend to mount an external disk from inside the container but have it be visible outside.

In order for new mounts from within a container to be propagated, you must set the following on the container:

1. `rootfsPropagation: shared`
2. The mount point into the container below which new mounts are to occur must be `rshared,rbind`. In practice, this is `/var` (or some subdir of `/var`), since that is the only true read-write area of the filesystem where you will mount things.

Thus, if you have a regular container that is only reading and writing, go ahead and do:

```yml
binds:
 - /var:/some/var/path
```

On the other hand, if you have a container that will make new mounts that you wish to be visible outside the container, do:

```yml
binds:
 - /var:/var:rshared,rbind
rootfsPropagation: shared
```
