This directory contains scripts/files to bootstrap a `etcd` cluster.

In the local, hyperkit based, setup, we use a `etcd` running in a
Docker for Mac container to bootstrap the cluster. For a cloud based demo, we'd use `https://discovery.etcd.io`. The host/DfM side is setup with [dfm-setup.sh](./dfm-setup.sh).

The moby `etcd` package is build with [build-pkg.sh](./build-pkg.sh). It take the official `etcd` container and adds a [script](./etcd.sh) to start `etcd`.

To run (for now):

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
