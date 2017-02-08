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

## Build a moby image

In the `alpine` subdirectory:
```shell
make gce
```
or (for a 4.4 kernel):
```shell
make LTS4.4=1 gce
```
You'll end up with `gce.img.tar.gz`. It's best to rename it to include the kernel version and the short commit tag or similar before uploading to GCP.

If you don't need/want to compile Moby from source, you can do a `make get` in the top-level directory before `make gce`. This downloads most of the build artefacts like the kernel and initrd based on your git hash.

## Upload an image and create a compute image

The next step is to upload the image and create a Compute image from it.

```shell
TARBALL=gce.img-4.9-cb44fd1.tar.gz
gsutil cp -a public-read ${TARBALL} gs://${NAME}/${TARBALL}
# Note, GCP does not like "." in images names
gcloud compute images create --source-uri \
  https://storage.googleapis.com/rolf/${TARBALL} moby-4-9-cb44fd1
```

## Create an instance and connect to it

With the image create, we can now create an instance and connect to
the serial port.

```shell
gcloud compute instances create my-node-4-9 \
  --image="moby-4-9-cb44fd1" --metadata serial-port-enable=true \
  --machine-type="g1-small" --boot-disk-size=200

gcloud compute connect-to-serial-port my-node-4-9
```
