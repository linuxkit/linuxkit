### swarmd package

This adds a `swarmd` package for Moby which contains the standalone
swarmkit orchestration daemon (`swarmd`) and CLI tool (`swarmctl`).

The package tracks [ijc's `container-wip` branch][containerd-wip].
Compared with mainline swarmkit (which container a basic containerd
executor merged in [PR1965]) this reworks the executor to use the
container client library and adds support for CNI networking.

With a suitable LinuxKit image (such as `swarmd.yml` from this
directory) something like this should work:

    ctr exec -- swarmd swarmd swarmctl service create --image docker.io/library/nginx:alpine --name nginx
    ctr exec -- swarmd swarmd swarmctl service ls

Note that `swarmd` uses the "swarmd" containerd namespace, so to see
swarmd managed containers you will need to use `-n swarmd` on all
`ctr` commands e.g.:

    ctr -n swarmd containers ls

Alternatively you may export `CONTAINERD_NAMESPACE=swarmd`.

### TODO

Bootstrapping a cluster needs more investigation. Tokens and join
addresses can currently only be passed on the `swarmd` command line
which is inconvenient for automated image deployment.

Swarmkit [PR 1965][PR1965] also contains a number of TODOs which are not
separately listed here.

[PR1965]: https://github.com/docker/swarmkit/pull/1965
[containerd-wip]: https://github.com/ijc/swarmkit/tree/containerd-wip
