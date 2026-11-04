# Docker contexts and container backends

LinuxKit uses Docker for package builders, optional Docker image-cache reads and
writes, and some output-format helper containers. These operations use the same
context-selection code in `src/cmd/linuxkit/dockercontext`. LinuxKit does not
use or change your Docker CLI's saved current context for context selection.

## General context selection

For Docker operations, LinuxKit selects the first applicable setting:

1. The global `linuxkit --context <name>` flag (also accepted after subcommands).
2. `DOCKER_CONTEXT`.
3. `DOCKER_HOST`, through Docker's special `default` context.
4. An existing context named `desktop-linux`.
5. Docker's special context named `default`.

`DOCKER_CONFIG` selects the Docker configuration directory; otherwise LinuxKit
uses `~/.docker`. Named contexts retain their endpoint, TLS certificates, and SSH
connection settings. Explicit flags take precedence over environment variables,
and `DOCKER_CONTEXT` takes precedence over `DOCKER_HOST`.

The saved `currentContext` setting from `docker context use` is ignored. Switching
that setting, including from another tool, does not redirect LinuxKit operations.
Without overrides, LinuxKit checks whether `desktop-linux` exists, then uses
`default` only if it is absent. If `desktop-linux` exists but its daemon is stopped,
unreachable, or misconfigured, LinuxKit reports that failure; it does not switch
to `default`. Use `--context default` to bypass `desktop-linux` explicitly.

The context **named `default` is not an alias for your current context**. Choosing
`--context default` deliberately uses Docker's environment-based default endpoint,
including `DOCKER_HOST` and TLS settings. Without `DOCKER_HOST`, this is Docker's
platform default socket or named pipe. It need not exist on your installation.
LinuxKit does not probe other runtimes or switch daemons if the selected endpoint
is unavailable.

For example:

```sh
# Use desktop-linux if that context exists; otherwise use default.
linuxkit build --docker image.yml

# Override it for all Docker operations in this command.
linuxkit --context desktop-linux build --docker image.yml

# Override it for this process using the environment.
DOCKER_CONTEXT=orbstack linuxkit pkg build pkg/example
```

Every Docker subprocess receives an explicit `--context` and `--config`. Docker
SDK operations use the same selected context rather than assuming a socket path.
Helper image pulls and runs use the same context, as do builder creation, config
and certificate copying, startup, and BuildKit connections.

## Package builders and image output

Package builders can use different contexts for different target platforms. The
selection order for each platform is:

1. Its entry in `--builders <platform>=<context>,...`.
2. Its entry in `LINUXKIT_BUILDERS`.
3. The global `--context` flag.
4. `DOCKER_CONTEXT`.
5. `DOCKER_HOST`, through the `default` context.
6. An existing context named `linuxkit-<platform>`, replacing `/` with `-`, such as
   `linuxkit-linux-arm64`.
7. An existing context named `desktop-linux`.
8. The literal context named `default`.

Flag mappings merge with environment mappings per platform: specifying one
platform on the command line does not clear mappings for other platforms.
General explicit and environment overrides suppress automatic `linuxkit-*`
discovery for platforms without mappings.

Once a context is selected, a connection, permission, configuration, or builder
startup failure is an error. Only an absent automatically discovered context falls
through to the next candidate: `linuxkit-<platform>`, then `desktop-linux`, then
`default`. An existing but broken discovered context does not trigger a build on
a different daemon. The same rules apply to
`linuxkit pkg builder du` and `linuxkit pkg builder prune`.

`linuxkit pkg build --docker` loads output images into the **general context**,
which can differ from the platform builder contexts. For example:

```sh
linuxkit --context colima pkg build --docker \
  --platforms linux/arm64,linux/amd64 \
  --builders linux/arm64=remote-arm64 pkg/example
```

This builds arm64 on `remote-arm64`, builds amd64 on `colima`, and loads the output
images into `colima`. Registry pulls and pushes through LinuxKit's own OCI cache
are independent of Docker daemon selection.

## Backends

### Docker Desktop on macOS and Windows

With no explicit overrides, LinuxKit uses `desktop-linux` if that context exists,
then `default` if it does not. You can select a context explicitly with `--context`.
On Windows, use `docker context ls` to identify the context for the desired engine.
LinuxKit package builders and output helpers require a Linux container engine.

There is no requirement to create `/var/run/docker.sock` or a compatibility
symlink merely to make context selection work. Selecting `default` explicitly can
still target a missing socket even while Docker Desktop is running.

### Colima

Use Colima's Docker runtime and select it with `--context colima` or
`DOCKER_CONTEXT=colima`.
Named Colima profiles can have other context names; obtain them with `docker
context ls`. Colima's containerd runtime is not a Docker endpoint.

### OrbStack

Select OrbStack with `--context orbstack` or `DOCKER_CONTEXT=orbstack`. LinuxKit
uses the context's socket and does not depend on a `/var/run/docker.sock` symlink.
For both Colima and OrbStack, `docker context use` alone does not select the
backend for LinuxKit.

### Podman

Use the Docker CLI connected to Podman's Docker-compatible API. Configure a Docker
context for the socket exposed by your Podman installation, or use `DOCKER_HOST`:

```sh
docker context create podman --docker "host=<your-podman-socket-URL>"
linuxkit --context podman pkg build pkg/example

# Alternative: use the socket URL directly; unset DOCKER_CONTEXT first if set.
DOCKER_HOST=<your-podman-socket-URL> linuxkit pkg build pkg/example
```

LinuxKit invokes a `docker` executable directly; a shell alias such as
`alias docker=podman` does not affect these subprocesses. A reachable compatibility
API alone does not guarantee package-builder support. Builders also require
privileged BuildKit containers, named volumes, Docker-compatible inspect and copy
operations, and `exec buildctl dial-stdio`. Rootless Podman or host security policy
can restrict these capabilities. Failures report the chosen context and daemon
error instead of redirecting to Docker Desktop or another installed runtime.

## Troubleshooting

Start with:

```sh
docker context ls
docker --context <name> info
linuxkit --context <name> -v=2 pkg build pkg/example
```

Check `DOCKER_CONTEXT`, `DOCKER_HOST`, `DOCKER_CONFIG`, and `LINUXKIT_BUILDERS` if
LinuxKit selects an unexpected endpoint. Debug logging includes the context and
selection source. Package dry runs also show the chosen builder context; they do
not connect to the daemon or validate builder capabilities.

Docker command errors retain the command, context, exit error, and stderr, even
when normal command output is suppressed. For example, a missing socket error
identifies both the selected context and the socket reported by Docker. Builder
readiness timeouts retain the last connection error.

`linuxkit build --docker` treats Docker as an optional image-cache source. A
missing image falls through to the LinuxKit cache and registry. A failed Docker
connection also permits that fallback, but emits a warning with the context and
underlying error. Registry-only builds and output formats that do not need Docker
continue to work without a Docker daemon.

## Maintaining context-aware code

Resolve general operations with `dockercontext.Resolve(explicitContext)` and
platform builders with `dockercontext.ResolveBuilder(mappedContext,
explicitContext, platform)`. Pass the resulting selection through operations on
that daemon. Use `Selection.Run` for Docker subprocesses, `Selection.Client` for
SDK access, and `Selection.BuildKitClient` for BuildKit connections. Keep explicit
context options in library APIs; do not implement new environment or current
context fallback logic at individual call sites. Do not substitute `default` for
an unspecified context or discard an error when selecting a fallback.
