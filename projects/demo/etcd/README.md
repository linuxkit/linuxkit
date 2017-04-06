This directory contains scripts/files to bootstrap a `etcd` cluster.

In the local, hyperkit based, setup, we use a `etcd` running in a
Docker for Mac container to bootstrap the cluster. For a cloud based demo, we'd use `https://discovery.etcd.io`. The host/DfM side is setup with [dfm-setup.sh](./dfm-setup.sh).

The moby `etcd` package is build with [build-pkg.sh](./build-pkg.sh). It take the official `etcd` container and adds a [script](./etcd.sh) to start `etcd`.


## Simple single node cluster (OUTDATED)

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

## InfraKit cluster setup (OUTDATED)

This should create a HyperKit based, InfraKit managed `etcd` cluster with 5 `etcd` instances.

#### Infrakit setup
You need the [infrakit](https://github.com/docker/infrakit) binaries for this. I normally compile from source using `make build-in-container`. The below was tried with commit `cb420e3e50ea60afe58538b1d3cab1cb14059433`.

- Make sure you start from scratch
```
rm -rf ~/.infrakit
```
- Start the infrakit plugins, each in it's own window from the root of the infrakit source tree:
```
infrakit-group-default
```
```
infrakit-flavor-vanilla
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
infrakit group commit infrakit.json
```

To check if everything is fine, note down the IP address from one of
the nodes and then:
```
docker run --rm -t quay.io/coreos/etcd:v3.1.5 etcdctl --endpoints http://192.168.65.24:2379 member list
```

## Infrakit GCP setup

Note: This setup is somewhat specific to our GCP setup (IP addresses
and account info) and needs to be adjusted to your setting. The
configuration is documented in the top-level README.md.

Build the image and upload it:
```
moby build etcd
```

Start the infrakit components in separate windows:
```
infrakit-group-default
infrakit-flavor-vanilla
infrakit-instance-gcp
```

Commit the configuration:
```
infrakit group commit infrakit-gce.json
```
