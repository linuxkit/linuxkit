# Using LinuxKit on Google Cloud Platform (GCP)

This is a quick guide to run LinuxKit on GCP. A lot of internal development and CI
has used Google Cloud so the support is very good; other platforms will have similar support soon.

## Setup

You have two choices for authentication with Google Cloud

1. You can use [Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials)
2. You can use a Service Account

### Application Default Credentials

You need the [Google Cloud SDK](https://cloud.google.com/sdk/)
installed.  Either install it from the URL or view `brew` (on a Mac):
```shell
brew tap caskroom/cask
brew cask install google-cloud-sdk
```

Or via source code:

```shell
curl -SsL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-151.0.0-darwin-x86_64.tar.gz
tar xzvf google-cloud-sdk-151.0.0-darwin-x86_64.tar.gz
./google-cloud-sdk/install.sh
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
gcloud auth application-default login
```

### Service Account

You can use [this guide](https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances#createanewserviceaccount)
to create a Service Account.

Make sure to download the credentials in JSON format and store them somewhere safe.

## Build an image

When using `linuxkit build ...` to build an image, specify `-format gcp` to
build an image in a format that GCP will understand. For example:

```
linuxkit build -format gcp myprefix.yml
```

This will create a local `myprefix.img.tar.gz` compressed image file.

## Push image

Do `linuxkit push gcp -project myproject-1234 -bucket bucketname myprefix.img.tar.gz` to upload it to the
specified bucket, and create a bootable image from the stored image.

Alternatively, you can set the project name and the bucket name using environment variables, `CLOUDSDK_CORE_PROJECT` and `CLOUDSDK_IMAGE_BUCKET`.
See the constant values defined in [`src/cmd/linuxkit/run_gcp.go`](../src/cmd/linuxkit/run_gcp.go) for the complete list of the supported environment variables.

## Create an instance and connect to it

With the image created, we can now create an instance and connect to
the serial port.

```
linuxkit run gcp -project myproject-1234 myprefix
```

## Nested Virtualization

Google Cloud offers [Nested
Virtualization](https://cloud.google.com/compute/docs/instances/enable-nested-virtualization-vm-instances)
as a beta feature. `linuxkit` supports this by pushing the image with
`linuxkit push gcp -nested-virt <other options>` and `linuxkit run gcp
-nested-virt <other options>`. The `push` sets the appropriate license
on the image while the `run` argument ensures that the CPU is at least
Haswell or newer.

