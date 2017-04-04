This directory contains scripts/files to bootstrap a `etcd` cluster.

In the local, hyperkit based, setup, we use a `etcd` running in a
Docker for Mac container to bootstrap the cluster. For a cloud based demo, we'd use `https://discovery.etcd.io`. The host/DfM side is setup with [dfm-setup.sh](./dfm-setup.sh).

The moby `etcd` package is build with [build-pkg.sh](./build-pkg.sh). It take the official `etcd` container and adds a [script](./etcd.sh) to start `etcd`.


## Simple single node cluster

- Edit `./dfm-setup.sh` and set `NUMPEERS` to `1`
- Start the etcd bootstrap container in on window:
```
./dfm-setup.sh
```

- In another window build/run the moby image:
```
./build-pkg.sh
moby build etcd
moby run etcd
```

## InfraKit cluster setup

This should create a HyperKit based, InfraKit managed `etcd` cluster with 5 `etcd` instances.

#### Infrakit setup
You need the [infrakit](https://github.com/docker/infrakit) binaries for this. I normally compile from source using `make build-in-container`. The below was tried with commit `2153cbb0c28d450d271bbcdb9b3765eb486a9ac9`

- Make sure you start from scratch
```
rm -rf ~/.infrakit
```
- Start the infrakit plugins, each in it's own window from the root of the infrakit source tree:
```
./build/infrakit-group-default
```
```
./build/infrakit-flavor-vanilla
```
- Start the hyperkit instance plugin from this directory:
```
../../../bin/infrakit-instance-hyperkit
```

#### etcd setup

- Start the bootstrap `etcd`:
```
./dfm-setup.sh
```

- Commit the infrakit config:
```
~/src/docker/infrakit/build/infrakit group commit infrakit.json
```

To check if everything is fine, note down the IP address from one of
the nodes and then:
```
docker run --rm -t quay.io/coreos/etcd:v3.1.5 etcdctl --endpoints http://192.168.65.24:2379 member list
```
