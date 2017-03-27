# Using Moby on Google Cloud Platform (GCP)

This is a quick guide to run Moby on GCP.

## Setup

You need the [Google Cloud SDK](https://cloud.google.com/sdk/)
installed.  Either install it from the URL or view `brew` (on a Mac):
```shell
brew install google-cloud-sdk
```

Then, set up some environment variables (adjust as needed) and login:
```shell
export CLOUDSDK_CORE_PROJECT=<GCP project>
export CLOUDSDK_COMPUTE_ZONE=europe-west1-d
gcloud auth login
```

The authentication will redirect to a browser with Google login.

Also authenticate local applications with
```
gcloud beta auth application-default login
```

## Build a moby image

Add a `gcp` output line to your yaml config, see the example in `examples/gcp.yml`.

Then do `./bin/moby myfile.yml`

This will create a local `myfile.img.tar.gz` compressed image file, upload it to the
specified bucket, and create a bootable image.

## Create an instance and connect to it

With the image created, we can now create an instance and connect to
the serial port.

```shell
gcloud compute instances create my-node \
  --image="myfile" --metadata serial-port-enable=true \
  --machine-type="g1-small" --boot-disk-size=200

gcloud compute connect-to-serial-port my-node
```
