# Using Moby on Google Cloud Platform (GCP)

This is a quick guide to run Moby on GCP.

## Setup

You have two choices for authentication with Google Cloud

1. You can use [Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials)
2. You can use a Service Account

### Application Default Credentials

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
gcloud auth application-default login
```

### Service Account

You can use [this guide](https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances#createanewserviceaccount)
to create a Service Account.

Make sure to download the credentials in JSON format and store them somewhere safe.

## Build a moby image

Add a `gcp` output line to your yaml config, see the example in `examples/gcp.yml`.

Then do `./bin/moby myfile.yml`

This will create a local `myfile.img.tar.gz` compressed image file, upload it to the
specified bucket, and create a bootable image.

## Create an instance and connect to it

With the image created, we can now create an instance and connect to
the serial port.

```
moby run gcp -project myproject-1234 myfile
```
