This directory contains scripts/files to bootstrap a `etcd` cluster both on the local machine as well as on Google Cloud.

An `etcd` cluster can be bootstrapped in different ways (see the [Documentatiob](https://coreos.com/etcd/docs/latest/op-guide/clustering.html) for more details. For the demo we use configuration via static IP addresses. With Infrakit these are managed by assigning `LogicalID`s to cluster members. The `LogicalID` is interpreted as a IP address.

The moby `etcd` package is build with [build-pkg.sh](./build-pkg.sh). It takes the official `etcd` container and adds a [script](./etcd.sh) to start `etcd`. [etcd.sh](./etcd.sh) first attempts to join a new cluster. If that fails it attempts to join an existing cluster. Note, the number and members of the cluster are somewhat hard coded in the script.


## Preparation

- Build the `etcd` image and then moby image:
```
./build-pkg.sh
moby build etcd
```

## InfraKit cluster setup (OUTDATED)

This should create a HyperKit based, InfraKit managed `etcd` cluster with 5 `etcd` instances.

Start InfraKit:
```
infrakit-flavor-vanilla &
infrakit-group-default &
../../../bin/infrakit-instance-hyperkit
```

Note: The HyperKit InfraKit plugin must be started from the directory
where the `etcd` mobylinux image is located.

Now, commit the new config:
```
infrakit group commit infrakit.json
```

To check if everything is fine, note down the IP address from one of
the nodes and then:
```
docker run --rm -t quay.io/coreos/etcd:v3.1.5 etcdctl --endpoints http://192.168.65.200:2379 member list
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
