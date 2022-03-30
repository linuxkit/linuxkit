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

## Console not displaying init or containerd output at boot

If you're not seeing `containerd` logs in the console during boot, make sure that your kernel `cmdline` configuration doesn't list multiple consoles.

`init` and other processes like `containerd` will use the last defined console in the kernel `cmdline`. When using `qemu`, to see the console you need to list `ttyS0` as the last console to properly see the output.

## Enabling and controlling containerd logs

On startup, linuxkit looks for and parses a file `/etc/containerd/runtime-config.toml`. If it exists, the content is used to configure containerd runtime.

Sample config is below:

```toml
cliopts="--log-level debug"
stderr="/var/log/containerd.out.log"
stdout="stdout"
```

The options are as follows:

* `cliopts`: options to pass to the containerd command-line as is.
* `stderr`: where to send stderr from containerd. If blank, it sends it to the default stderr, which is the console.
* `stdout`: where to send stdout from containerd. If blank, it sends it to the default stdout, which is the console. containerd normally does not have any stdout.

The `stderr` and `stdout` options can take exactly one of the following options:

* `stderr` - send to stderr
* `stdout` - send to stdout
* any absolute path (beginning with `/`) - send to that file. If the file exists, append to it; if not, create it and append to it.

Thus, to enable
a higher log level, for example `debug`, create a file whose contents are `--log-level debug` and place it on the image:

```yml
files:
  - path: /etc/containerd/runtime-config.toml
    source: "/path/to/runtime-config.toml"
    mode: "0644"
```

Note that the package that parses the `cliopts` splits on _all_ whitespace. It does not, as of this writing, support shell-like parsing, so the following will work:

```
--log-level debug --arg abcd
```

while the following will not:

```
--log-level debug --arg 'abcd def'
```

## Troubleshooting containers

Linuxkit runs all services in a specific `containerd` namespace called `services.linuxkit`. To list all the defined containers:

```sh
(ns: getty) linuxkit-befde23bc535:~# ctr -n services.linuxkit container ls
CONTAINER               IMAGE    RUNTIME
getty                   -        io.containerd.runtime.v1.linux
```

To list all running containers and their status:

```sh
(ns: getty) linuxkit-befde23bc535:~# ctr -n services.linuxkit task ls
TASK                    PID    STATUS
getty                   661    RUNNING
```

To list all processes running in a container:

```sh
(ns: getty) linuxkit-befde23bc535:/containers/services/getty# ctr -n services.linuxkit task ps getty
PID     INFO
661     &ProcessDetails{ExecID:getty,}
677     -
685     -
686     -
687     -
1237    -
```

To attach a shell to a running container:

```sh
(ns: getty) linuxkit-befde23bc535:/containers/services/getty# ctr -n services.linuxkit tasks exec --tty --exec-id sh sshd /bin/ash -l
(ns: sshd) linuxkit-befde23bc535:/#
```

Containers are defined as OCI bundles in `/containers`.
