This directory contains files used in moby demos.

# Prerequisites

Most of the scripts/files assume you are on a Mac.

- Recent Docker for Mac installed (edge, nightly, master channel)
- Currently, you need a *custom* version of VPNKit installed in Docker
  for Mac (see below)
- For the GCP portion: `brew install google-cloud-sdk`
- For `etcd`: `brew install etcd`
- Infrakit: Clone [infrakit](https://github.com/docker/infrakit) and
  the [GCP plugin](https://github.com/docker/infrakit.gcp) for
  infrakit.  For each, `make build-in-container` and then copy the
  contents of `./build` somewhere in your path.

For some of the demos, you currently need an updated version of VPNKit
for Docker for Mac. Hopefully this version will ship as default soon.

Quit docker for Mac
```
curl -fsSL --retry 10 -z vpnkit.tgz -o vpnkit.tgz https://circle-artifacts.com/gh/docker/vpnkit/708/artifacts/0/Users/distiller/vpnkit/vpnkit.tgz

tar xzvf vpnkit.tgz
cp Contents/MacOS/vpnkit /Applications/Docker.app/Contents/Resources/bin/
```
Restart Docker for Mac.


# Local setup

We use a `socat` container to forward ports from the VM to localhost (via Docker for Mac), to make it easier to access some VMs. To build
```
(cd dockerfiles; docker build -t socat -f Dockerfile.socat .)
```
And then run:
```
docker run --rm -t -d -p 8080:8080 socat tcp-listen:8080,reuseaddr,fork tcp:192.168.65.100:80
```
This forwards local port `8080` to `192.168.65.100:80`, so if you start, say the `intro` image, run `moby run -ip 196.168.65.100 intro`


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


