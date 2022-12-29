#!/bin/sh
# SUMMARY: Check that gcp image boots in gcp
# LABELS: skip

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=gcp-$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 8 | head -n 1)
export CLOUDSDK_CORE_PROJECT="moby-datakit-ci"
export CLOUDSDK_COMPUTE_ZONE="europe-west1-d"
export CLOUDSDK_IMAGE_BUCKET="linuxkit-gcp-test-bucket"

clean_up() {
	rm -rf ${NAME}*
    docker run -i --rm \
        -e CLOUDSDK_CORE_PROJECT \
        -v `pwd`/certs:/certs \
        google/cloud-sdk \
        sh -c "gcloud auth activate-service-account --key-file /certs/svc_account.json; \
        gsutil rm gs://${CLOUDSDK_IMAGE_BUCKET}/${NAME}.img.tar.gz" || true
    rm -rf certs
}
trap clean_up EXIT

[ -n "$GCLOUD_CREDENTIALS" ] || exit 1
mkdir -p certs
printf '%s' "$GCLOUD_CREDENTIALS" > certs/svc_account.json

linuxkit build --format gcp --name "${NAME}" test.yml
[ -f "${NAME}.img.tar.gz" ] || exit 1
linuxkit push gcp -keys certs/svc_account.json -bucket linuxkit-gcp-test-bucket ${NAME}.img.tar.gz
# tee output of lk run to file as grep hides failures and doesn't 
# always allow the vm to be cleaned up
linuxkit run gcp -keys certs/svc_account.json ${NAME} | tee ${NAME}.log
grep -q "Welcome to LinuxKit" ${NAME}.log

exit 0