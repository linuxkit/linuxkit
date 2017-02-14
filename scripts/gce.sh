#!/bin/bash
#
# TODO:
# - Use docker repo google/cloud-sdk as per Alpine/makefile upload-gce:
# - Handle race conditions between boot and serial connection
#
# This script
# - Uploads gce-test.img.tar.gz
# - Creates an image
# - Launches instances
# - Collects output from serial port
# - Deletes instance, image and uploaded object
#
# Images and instances are named with a combination of the current git commit
# hash and and a random number.  The random number is to avoid collisions
# if mutliple people are using the same GCP project. Example:
#
#    img-inst-069b999-dirty-u3d5c1a7a
#             ^git hash      ^random
# Pre-reqs:
# - Override env variables below for project, zone and bucket
# - Install gcloud, eg::
#      brew install gcloud
# - Authenticate:
#      gclould auth login
#
# Set env variable INTERACTIVE=1 for interactive shell via
# serial port
#
#
set -e
#set -x

# Override CLOUDSDK_* and BUCKET by setting environment
# variables or ask to be added to Moby-CI project
: ${CLOUDSDK_CORE_PROJECT:="moby-ci"}
: ${CLOUDSDK_COMPUTE_ZONE:="us-central1-c"}
: ${BUCKET:="com-docker-moby-ci"}
export CLOUDSDK_CORE_PROJECT CLOUDSDK_COMPUTE_ZONE BUCKET

if [[ -n $1 ]]; then
    LOG=$1
else
    LOG=test.log
fi

git status -s  &> /dev/null && DIRTY="-dirty"
GITHASH=$(git rev-parse --short HEAD)${DIRTY}

UNIQ=${GITHASH}-u"$(printf '%x%x' $RANDOM $RANDOM)"

: ${TARBALL="alpine/gce-test.img.tar.gz"}
GSOBJ="gce-test.img-${UNIQ}.tar.gz"
GSOBJ_URL="https://storage.googleapis.com/${BUCKET}/${GSOBJ}"
IMG_NAME="img-${UNIQ}"
INST_NAME="inst-${UNIQ}"

cleanup () {
  set +e
  echo "Cleaning up..."
  gcloud compute -q instances delete ${INST_NAME}
  gcloud compute -q images delete ${IMG_NAME}
  gsutil rm gs://${BUCKET}/${GSOBJ}
}
trap cleanup exit

echo "Uploading ${TARBALL} to bucket..."
gsutil cp ${TARBALL} gs://${BUCKET}/${GSOBJ}
echo "Creating GCE image from bucket..."
gcloud compute images create --source-uri ${GSOBJ_URL} ${IMG_NAME}
echo "Creating GCE instance..."
gcloud compute instances create ${INST_NAME} \
    --image=${IMG_NAME} \
    --metadata serial-port-enable=true \
    --machine-type="n1-standard-1" \
    --boot-disk-size=200

if [[ -n ${INTERACT} ]]; then
  echo "Interactive session..."
  gcloud compute connect-to-serial-port ${INST_NAME}
else
  # This works because Moby test shuts moby down.
  echo "Tailing serial port buffer until shutdown..."
  gcloud compute instances tail-serial-port-output ${INST_NAME} | tee ${LOG} && true
fi
