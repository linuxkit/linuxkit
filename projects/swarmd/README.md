### swarmd package

This adds a `swarmd` package for Moby which contains the standalone
swarmkit orchestration daemon (`swarmd`) and CLI tool (`swarmctl`).

The package tracks [docker/swarmkit#1965][PR1965] which
is a WIP PR adding a containerd executor to swarmkit.

With a suitable moby image (such as `swarmd.yml` from this directory)
something like this should work:

    runc exec swarmd swarmctl service create --image docker.io/library/nginx:alpine --name nginx
    runc exec swarmd swarmctl service ls

### TODO

Currently the swarm state directory needs to be at a path which is
identical from the PoV of both the `containerd` and `swarmd`
processes. For now this means that the swarmkit state is put in
`/var/lib/containerd/swarmd`.

Bootstrapping a cluster needs more invesigation. Tokens and join
addresses can currently only be passed on the `swarmd` command line
which is inconvenient for automated image deployment.

Swarmkit [PR 1965][PR1965] also contains a number of TODOs which are not
separately listed here.

[PR1665]: https://github.com/docker/swarmkit/pull/1965
