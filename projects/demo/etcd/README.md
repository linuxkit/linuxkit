This directory contains scripts/files to bootstrap a `etcd` cluster both on the local machine as well as on Google Cloud.

An `etcd` cluster can be bootstrapped in different ways (see the [Documentation](https://coreos.com/etcd/docs/latest/op-guide/clustering.html) for more details. For the demo we use configuration via static IP addresses. With Infrakit these are managed by assigning `LogicalID`s to cluster members. The `LogicalID` is interpreted as a IP address.

The moby `etcd` package is build with [build-pkg.sh](./build-pkg.sh). It takes the official `etcd` container and adds a [script](./etcd.sh) to start `etcd`. [etcd.sh](./etcd.sh) first attempts to join a new cluster. If that fails it attempts to join an existing cluster. Note, the number and members of the cluster are somewhat hard coded in the script.

Each node is also configured with a disk, which is mounted inside the
`etcd` container. `etcd` uses it to keep some state to help with
restarts.

## Preparation

- Build the `etcd` image and then moby image inside the `pkg` directory:
```
./build-pkg.sh
moby build etcd
```

## InfraKit cluster setup

This should create a HyperKit based, InfraKit managed `etcd` cluster with 5 `etcd` instances.

Start InfraKit:
```
./start-infrakit
```

Note: The HyperKit InfraKit plugin must be started from the directory
where the `etcd` mobylinux image is located.

Now, commit the new config:
```
infrakit group commit infrakit.json
```

To check if everything is fine, we created (above) a local `etcd.local` docker image which already has the environment set up to contact the cluster:
```
docker run --rm -ti etcd.local etcdctl member list
docker run --rm -ti etcd.local etcdctl cluster-health
```

You can perform rolling updates, by for example, switching the kernel version in `etcd.yml`, build a new moby, e.g., `moby build -name etcd-4.10 etcd`, update `infrakit.json`, and then commit the new configuration to InfraKit: `infrakit group commit infrakit.json`.


## Infrakit GCP setup

You need to do the general setup as described in the demo [README](../README.md). Specifically, you need the `CLOUDSDK_*` environment variables set and you need to have authenticated with GCP.

Note, the demo uses static IP addresses and they are specific to our
setup. The IP addresses need to be changed in the `infrakit-gcp.json`
config file.

In order to use the static IP addresses we created a custom network:
```
gcloud compute networks create rneugeba-demo --mode auto
gcloud compute networks subnets list
# get IP subnet for rneugeba-demo
gcloud compute firewall-rules create rneugeba-demo-internal --network \
    rneugeba-demo --allow tcp,udp,icmp --source-ranges 10.132.0.0/9
```
The firewall setup means that all our projects networks can talk to the demo
network.


Build the image and upload it:
```
moby build etcd
```

Start infrakit as above:
```
./start-infrakit
```

Commit the configuration:
```
infrakit group commit infrakit-gcp.json
```
