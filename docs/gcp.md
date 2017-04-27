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

Add a `gcp-img` output line to your yaml config, see the example in `examples/gcp.yml`.

Then do `moby build myfile.yml`

This will create a local `myfile.img.tar.gz` compressed image file.

## Push image

Do `moby push gcp -project myproject-1234 -bucket bucketname myfile.img.tar.gz` to upload it to the
specified bucket, and create a bootable image from the stored image.

## Create an instance and connect to it

With the image created, we can now create an instance and connect to
the serial port.

```
moby run gcp -project myproject-1234 myfile
```
