# Configuration Reference

The `linuxkit build` command assembles a set of containerised components into in image. The simplest
type of image is just a `tar` file of the contents (useful for debugging) but more useful
outputs add a `Dockerfile` to build a container, or build a full disk image that can be
booted as a linuxKit VM. The main use case is to build an assembly that includes
`containerd` to run a set of containers, but the tooling is very generic.

The yaml configuration specifies the components used to build up an image . All components
are downloaded at build time to create an image. The image is self-contained and immutable,
so it can be tested reliably for continuous delivery.

Components are specified as Docker images which are pulled from a registry during build if they
are not available locally. See [image-cache](./image-cache.md) for more details on local caching.
The Docker images are optionally verified with Docker Content Trust.
For private registries or private repositories on a registry credentials provided via
`docker login` are re-used.

The configuration file is processed in the order `kernel`, `init`, `onboot`, `onshutdown`,
`services`, `files`. Each section adds files to the root file system. Sections may be omitted.

Each container that is specified is allocated a unique `uid` and `gid` that it may use if it
wishes to run as an isolated user (or user namespace). Anywhere you specify a `uid` or `gid`
field you specify either the numeric id, or if you use a name it will refer to the id allocated
to the container with that name.

```
services:
  - name: redis
    image: redis:latest
    uid: redis
    gid: redis
    binds:
     - /etc/redis:/etc/redis
files:
  - path: /etc/redis/redis.conf
    contents: "..."
    uid: redis
    gid: redis
    mode: "0600"
```

## `kernel`

The `kernel` section is only required if booting a VM. The files will be put into the `boot/`
directory, where they are used to build bootable images.

The `kernel` section defines the kernel configuration. The `image` field specifies the Docker image,
which should contain a `kernel` file that will be booted (eg a `bzImage` for `amd64`) and a file
called `kernel.tar` which is a tarball that is unpacked into the root, which should usually
contain a kernel modules directory. `cmdline` specifies the kernel command line options if required.

To override the names, you can specify the kernel image name with `binary: bzImage` and the tar image
with `tar: kernel.tar` or the empty string or `none` if you do not want to use a tarball at all.

Kernel packages may also contain a cpio archive containing CPU microcode which needs prepending to
the initrd. To select this option, recommended when booting on bare metal, add `ucode: intel-ucode.cpio`
to the kernel section.

## `init`

The `init` section is a list of images that are used for the `init` system and are unpacked directly
into the root filesystem. This should bring up `containerd`, start the system and daemon containers,
and set up basic filesystem mounts. in the case of a LinuxKit system. For ease of
modification `runc` and `containerd` images, which just contain these programs are added here
rather than bundled into the `init` container.

## `onboot`

The `onboot` section is a list of images. These images are run before any other
images. They are run sequentially and each must exit before the next one is run.
These images can be used to configure one shot settings. See [Image
specification](#image-specification) for a list of supported fields.

## `onshutdown`

This is a list of images to run on a clean shutdown. Note that you must not rely on these
being run at all, as machines may be be powered off or shut down without having time to run
these scripts. If you add anything here you should test both in the case where they are
run and when they are not. Most systems are likely to be "crash only" and not have any setup here,
but you can attempt to deregister cleanly from a network service here, rather than relying
on timeouts, for example.

## `services`

The `services` section is a list of images for long running services which are
run with `containerd`.  Startup order is undefined, so containers should wait
on any resources, such as networking, that they need.  See [Image
specification](#image-specification) for a list of supported fields.

## `files`

The files section can be used to add files inline in the config, or from an external file.

```
files:
  - path: dir
    directory: true
    mode: "0777"
  - path: dir/name1
    source: "/some/path/on/local/filesystem"
    mode: "0666"
  - path: dir/name2
    source: "/some/path/that/it/is/ok/to/omit"
    optional: true
    mode: "0666"
  - path: dir/name3
    contents: "orange"
    mode: "0644"
    uid: 100
    gid: 100
```

Specifying the `mode` is optional, and will default to `0600`. Leading directories will be
created if not specified. You can use `~/path` in `source` to specify a path in the build
user's home directory.

In addition there is a `metadata` option that will generate the file. Currently the only value
supported here is `"yaml"` which will output the yaml used to generate the image into the specified
file:
```
  - path: etc/linuxkit.yml
    metadata: yaml
```

Note that if you use templates in the yaml, the final resolved version will be included in the image,
and not the original input template.

Because a `tmpfs` is mounted onto `/var`, `/run`, and `/tmp` by default, the `tmpfs` mounts will shadow anything specified in `files` section for those directories.

## Image specification

Entries in the `onboot` and `services` sections specify an OCI image and
options. Default values may be specified using the `org.mobyproject.config` image label.
For more details see the [OCI specification](https://github.com/opencontainers/runtime-spec/blob/master/spec.md).

If the `org.mobylinux.config` label is set in the image, that specifies default values for these fields if they
are not set in the yaml file. While most fields are _replaced_ if they are specified in the yaml file,
some support _add_ via the format `<field>.add`; see below.
You can override the label entirely by setting the value, or setting it to be empty to remove
the specification for that value in the label.

If you need an OCI option that is not specified here please open an issue or pull request as the list is not yet
complete.

By default the containers will be run in the host `net`, `ipc` and `uts` namespaces, as that is the usual requirement;
in many ways they behave like pods in Kubernetes. Mount points must already exist, as must a file or directory being
bind mounted into a container.

- `name` a unique name for the program being executed, used as the `containerd` id.
- `image` the Docker image to use for the root filesystem. The default command, path and environment are
  extracted from this so they need not be filled in.
- `capabilities` the Linux capabilities required, for example `CAP_SYS_ADMIN`. If there is a single
  capability `all` then all capabilities are added.
- `capabilities.add` the Linux capabilities required, but these are added to the defaults, rather than overriding them.
- `ambient` the Linux ambient capabilities (capabilities passed to non root users) that are required.
- `mounts` is the full form for specifying a mount, which requires `type`, `source`, `destination`
  and a list of `options`. If any fields are omitted, sensible defaults are used if possible, for example
  if the `type` is `dev` it is assumed you want to mount at `/dev`. The default mounts and their options
  can be replaced by specifying a mount with new options here at the same mount point.
- `binds` is a simpler interface to specify bind mounts, accepting a string like `/src:/dest:opt1,opt2`
  similar to the `-v` option for bind mounts in Docker.
- `binds.add` is a simpler interface to specify bind mounts, but these are added to the defaults, rather than overriding them.
- `tmpfs` is a simpler interface to mount a `tmpfs`, like `--tmpfs` in Docker, taking `/dest:opt1,opt2`.
- `command` will override the command and entrypoint in the image with a new list of commands.
- `env` will override the environment in the image with a new environment list. Specify variables as `VAR=value`.
- `cwd` will set the working directory, defaults to `/`.
- `net` sets the network namespace, either to a path, or if `none` or `new` is specified it will use a new namespace.
- `ipc` sets the ipc namespace, either to a path, or if `new` is specified it will use a new namespace.
- `uts` sets the uts namespace, either to a path, or if `new` is specified it will use a new namespace.
- `pid` sets the pid namespace, either to a path, or if `host` is specified it will use the host namespace.
- `readonly` sets the root filesystem to read only, and changes the other default filesystems to read only.
- `maskedPaths` sets paths which should be hidden.
- `readonlyPaths` sets paths to read only.
- `uid` sets the user id of the process.
- `gid` sets the group id of the process.
- `additionalGids` sets a list of additional groups for the process.
- `noNewPrivileges` is `true` means no additional capabilities can be acquired and `suid` binaries do not work.
- `hostname` sets the hostname inside the image.
- `oomScoreAdj` changes the OOM score.
- `rootfsPropagation` sets the rootfs propagation, eg `shared`, `slave` or (default) `private`.
- `cgroupsPath` sets the path for cgroups.
- `resources` sets cgroup resource limits as per the OCI spec.
- `sysctl` sets a map of `sysctl` key value pairs that are set inside the container namespace.
- `rmlimits` sets a list of `rlimit` values in the form `name,soft,hard`, eg `nofile,100,200`. You can use `unlimited` as a value too.
- `annotations` sets a map of key value pairs as OCI metadata.

There are experimental `userns`, `uidMappings` and `gidMappings` options for user namespaces but these are not yet supported, and may have
permissions issues in use.

In addition to the parts of the specification above used to generate the OCI spec, there is a `runtime` section in the image specification
which specifies some actions to take place when the container is being started.
- `cgroups` takes a list of cgroups that will be created before the container is run.
- `mounts` takes a list of mount specifications (`source`, `destination`, `type`, `options`) and mounts them in the root namespace before the container is created. It will
  try to make any missing destination directories.
- `mkdir` takes a list of directories to create at runtime, in the root mount namespace. These are created before the container is started, so they can be used to create
  directories for bind mounts, for example in `/tmp` or `/run` which would otherwise be empty.
- `interface` defines a list of actions to perform on a network interface:
  - `name` specifies the name of an interface. An existing interface with this name will be moved into the container's network namespace.
  - `add` specifies a type of interface to be created in the containers namespace, with the specified name.
  - `createInRoot` is a boolean which specifes that the interface being `add`ed should be created in the root namespace first, then moved. This is needed for `wireguard` interfaces.
  - `peer` specifies the name of the other end when creating a `veth` interface. This end will remain in the root namespace, where it can be attached to a bridge. Specifying this implies `add: veth`.
- `bindNS` specifies a namespace type and a path where the namespace from the container being created will be bound. This allows a namespace to be set up in an `onboot` container, and then
  using `net: path` for a `service` container to use that network namespace later.
- `namespace` overrides the LinuxKit default containerd namespace to put the container in; only applicable to services.

An example of using the `runtime` config to configure a network namespace with `wireguard` and then run `nginx` in that namespace is shown below:
```
onboot:
  - name: dhcpcd
    image: linuxkit/dhcpcd:<hash>
    command: ["/sbin/dhcpcd", "--nobackground", "-f", "/dhcpcd.conf", "-1"]
  - name: wg
    image: linuxkit/ip:<hash>
    net: new
    binds:
      - /etc/wireguard:/etc/wireguard
    command: ["sh", "-c", "ip link set dev wg0 up; ip address add dev wg0 192.168.2.1 peer 192.168.2.2; wg setconf wg0 /etc/wireguard/wg0.conf; wg show wg0"]
    runtime:
      interfaces:
        - name: wg0
          add: wireguard
          createInRoot: true
      bindNS:
        net: /run/netns/wg
services:
  - name: nginx
    image: nginx:alpine
    net: /run/netns/wg
    capabilities:
     - CAP_NET_BIND_SERVICE
     - CAP_CHOWN
     - CAP_SETUID
     - CAP_SETGID
     - CAP_DAC_OVERRIDE
```

## `devices`

To access the console, it's necessary to explicitly add a "device" definition, for example:

```
devices:
- path: "/dev/console"
  type: c
  major: 5
  minor: 1
  mode: 0666
```

See the [getty package](../pkg/getty/build.yml) for a more complete example
and see [runc](https://github.com/opencontainers/runc/commit/60e21ec26e15945259d4b1e790e8fd119ee86467) for context.

To grant access to all block devices use:

```
devices:
- path: all
  type: b
```

See the [format package](../pkg/format/build.yml) for an example.

### Mount Options
When mounting filesystem paths into a container - whether as part of `onboot` or `services` - there are several options of which you need to be aware. Using them properly is necessary for your containers to function properly.

For most containers - e.g. nginx or even docker - these options are not needed. Simply doing the following will work fine:

```yml
binds:
 - /var:/some/var/path
```

Please note that `binds` doesn't **add** the mount points, but **replaces** them.
You can examine the `Dockerfile` of the component (in particular, `binds` value of
`org.mobyproject.config` label) to get the list of the existing binds.

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

## Templates

The `yaml` file supports templates for the names of images. Anyplace an image is used in a file and begins
with the character `@`, it indicates that it is not an actual name, but a template. The first word after
the `@` indicates the type of template, and the rest of the line is the argument to the template. The
templates currently supported are:

* `@pkg:` - the argument is the path to a linuxkit package. For example, `@pkg:./pkg/init`.

For `pkg`, linuxkit will resolve the path to the package, and then run the equivalent of `linuxkit pkg show-tag <dir>`.
For example:

```yaml
init:
  - "@pkg:../pkg/init"
```

Will cause linuxkit to resolve `../pkg/init` to a package, and then run `linuxkit pkg show-tag ../pkg/init`.

The paths are relative to the directory of the yaml file.
You can specify absolute paths, although it is not recommended, as that can make the yaml file less portable.

The `@pkg:` templating is supported **only** when the yaml file is being read from a local filesystem. It does not
support when using via stdin, e.g. `cat linuxkit.yml | linuxkit build -`, or URLs, e.g. `linuxkit build https://example.com/foo.yml`.

The `@pkg:` template currently supports only default `linuxkit pkg` options, i.e. `build.yml` and `tag` options. There
are no command-line options to override them.

**Note:** The character `@` is reserved in yaml. To use it in the beginning of a string, you must put the entire string in
quotes.

If you use the template, the actual derived value, and not the initial template, is what will be stored in the final
image when adding it via:

```yaml
files:
  - path: etc/linuxkit.yml
    metadata: yaml
```
