This directory contains files used in moby demos.

# Prerequisites

Most of the scripts/files assume you are on a Mac.

- Recent Docker for Mac installed (edge, nightly, master channel)
- For the GCP portion: `brew install google-cloud-sdk`
- For `etcd`: `brew install etcd`
- Infrakit: Clone [infrakit](https://github.com/docker/infrakit) and
  the [GCP plugin](https://github.com/docker/infrakit.gcp) for
  infrakit.  For each, `make build-in-container` and then copy the
  contents of `./build` somewhere in your path.

# GCP Setup

You probably want to change the project/zone
```
export CLOUDSDK_CORE_PROJECT=docker4x
export CLOUDSDK_COMPUTE_ZONE=europe-west1-d
gcloud auth login
gcloud auth application-default login
```

You may also want to create ssh-keys and upload them. See the [Generating a new SSH key-pair section](https://cloud.google.com/compute/docs/instances/connecting-to-instance)

One time configuration of the network:
```
gcloud compute networks create rneugeba-demo --mode auto
gcloud compute networks subnets list
# get IP subnet for rneugeba-demo
gcloud compute firewall-rules create rneugeba-demo-internal --network \
    rneugeba-demo --allow tcp,udp,icmp --source-ranges 10.128.0.0/9
```
The firewall setup means that all our projects networks can talk to the demo network.


