This directory contains files used in Moby/LinuxKit DockerCon 2017
keynote etcd cluster demo. They mostly serve as examples and probably
need adjustments to your specific environment. They may also break
over time :)

## Prerequisites

Most of the scripts/files assume you are on a Mac.

- Recent Docker for Mac installed (We used 17.05.0-ce-rc1-mac8 from the edge channel)
- For the GCP portion: `brew install google-cloud-sdk`
- Infrakit: Clone [infrakit](https://github.com/docker/infrakit) and
  the [GCP plugin](https://github.com/docker/infrakit.gcp) for
  infrakit.  The GCP plugin, needs to be v0.1. For each, `make
  build-in-container` and then copy the contents of `./build`
  somewhere in your path.

## etcd cluster setup

An `etcd` cluster can be bootstrapped in different ways (see the [Documentation](https://coreos.com/etcd/docs/latest/op-guide/clustering.html) for more details. For the demo we use configuration via static IP addresses. With Infrakit these are managed by assigning `LogicalID`s to cluster members. The `LogicalID` is interpreted as a IP address.

The `etcd` package takes the official `etcd` container and adds a
[script](./pkg/etcd.sh) to start `etcd`. [etcd.sh](./pkg/etcd.sh)
first attempts to join a new cluster. If that fails it attempts to
join an existing cluster. Note, the number and members of the cluster
are somewhat hard coded in the script.

Each node is also configured with a disk, which is mounted inside the
`etcd` container. `etcd` uses it to keep some state to help with
restarts.

## GCP Setup

You probably want to change the project/zone
```
export CLOUDSDK_CORE_PROJECT=docker4x
export CLOUDSDK_COMPUTE_ZONE=europe-west1-d
gcloud auth login
gcloud auth application-default login
```

You may also want to create ssh-keys and upload them. See the [Generating a new SSH key-pair section](https://cloud.google.com/compute/docs/instances/connecting-to-instance)

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

## Preparation

We create a number of local packages, not pulled from Hub. To build them, invoke `./build-pkg.sh` in the `./pkg` directory.

Then build the various YAML files using the `moby` tool and package/upload them to Google Cloud using the `linuxkit` tool.

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

Start infrakit as above:
```
./start-infrakit
```

Commit the configuration:
```
infrakit group commit infrakit-gcp.json
```

## Prometheus server

The etcd nodes use the Prometheus node exported. You can use the prometheus server image, also in this directory, to collect statistics from etc node. We currently build a specific Prometheus images with hard coded IP addresses. Ideally, the information should be passed in via the metadata/userdata.
