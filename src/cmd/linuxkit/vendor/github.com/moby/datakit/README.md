## DataKit -- Orchestrate applications using a Git-like dataflow

*DataKit* is a tool to orchestrate applications using a Git-like dataflow. It
revisits the UNIX pipeline concept, with a modern twist: streams of
tree-structured data instead of raw text. DataKit allows you to define
complex build pipelines over version-controlled data.

DataKit is currently used as the coordination
layer for [HyperKit](http://github.com/docker/hyperkit), the
hypervisor component of
[Docker for Mac and Windows](https://blog.docker.com/2016/03/docker-for-mac-windows-beta/), and
for the [DataKitCI][] continuous integration system.

---

[![Build Status (OSX, Linux)](https://travis-ci.org/moby/datakit.svg)](https://travis-ci.org/moby/datakit)
[![Build status (Windows)](https://ci.appveyor.com/api/projects/status/6qrdgiqbhi4sehmy/branch/master?svg=true)](https://ci.appveyor.com/project/moby/datakit/branch/master)
[![docs](https://img.shields.io/badge/doc-online-blue.svg)](https://docker.github.io/datakit/)

There are several components in this repository:

- `src` contains the main DataKit service. This is a Git-like database to which other services can connect.
- `ci` contains [DataKitCI][], a continuous integration system that uses DataKit to monitor repositories and store build results.
- `ci/self-ci` is the CI configuration for DataKitCI that tests DataKit itself.
- `bridge/github` is a service that monitors repositories on GitHub and syncs their metadata with a DataKit database.
  e.g. when a pull request is opened or updated, it will commit that information to DataKit. If you commit a status message to DataKit, the bridge will push it to GitHub.
- `bridge/local` is a drop-in replacement for `bridge/github` that just monitors a local Git repository. This is useful for local testing.

### Quick Start

The easiest way to use DataKit is to start both the server and the client in containers.

To expose a Git repository as a 9p endpoint on port 5640 on a private network, run:

```shell
$ docker network create datakit-net # create a private network
$ docker run -it --net datakit-net --name datakit -v <path/to/git/repo>:/data datakit/db
```

*Note*: The `--name datakit` option is mandatory.  It will allow the client
to connect to a known name on the private network.

You can then start a DataKit client, which will mount the 9p endpoint and
expose the database as a filesystem API:

```shell
# In an other terminal
$ docker run -it --privileged --net datakit-net datakit/client
$ ls /db
branch     remotes    snapshots  trees
```

*Note*: the `--privileged` option is needed because the container will have
to mount the 9p endpoint into its local filesystem.

Now you can explore, edit and script `/db`. See the
[Filesystem API][]
for more details.

### Building

The easiest way to build the DataKit project is to use [docker](https://docker.com),
(which is what the
[start-datakit.sh](https://github.com/moby/datakit/blob/master/scripts/start-datakit.sh) script
does under the hood):

```shell
docker build -t datakit/db -f Dockerfile .
docker run -p 5640:5640 -it --rm datakit/db --listen-9p=tcp://0.0.0.0:5640
```
These commands will expose the database's 9p endpoint on port 5640.

If you want to build the project from source without Docker, you will need to install
[ocaml](http://ocaml.org/) and [opam](http://opam.ocaml.org/). Then write:

```shell
$ make depends
$ make && make test
```

For information about command-line options:

```shell
$ datakit --help
```

## Prometheus metric reporting

Run with `--listen-prometheus 9090` to expose metrics at `http://*:9090/metrics`.

Note: there is no encryption and no access control. You are expected to run the
database in a container and to not export this port to the outside world. You
can either collect the metrics by running a Prometheus service in a container
on the same Docker network, or front the service with nginx or similar if you
want to collect metrics remotely.

## Language bindings

* **Go** bindings are in the `api/go` directory.
* **OCaml** bindings are in the `api/ocaml` directory. See `examples/ocaml-client` for an example.

## Licensing

DataKit is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/moby/datakit/blob/master/LICENSE.md) for the full
license text.

Contributions are welcome under the terms of this license. You may wish to browse
the [weekly reports](reports) to read about overall activity in the repository.

[DataKitCI]: https://github.com/moby/datakit/tree/master/ci
[Filesystem API]: https://github.com/moby/datakit/tree/master/9p.md
