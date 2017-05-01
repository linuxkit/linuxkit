#! /bin/sh

REPO="linuxkit/kernel-ubuntu"
BASE_URL=http://mirrors.kernel.org/ubuntu/pool/main/l/linux/

TAGS=$(curl --silent -f -lSL https://registry.hub.docker.com/v1/repositories/${REPO}/tags)

ARCH=amd64
LINKS=$(curl -s ${BASE_URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')
# Just get names for 4.x kernels
KERNELS=$(echo $LINKS | \
    grep -o "linux-image-4\.[0-9]\+\.[0-9]\+-[0-9]\+-generic_[^ ]\+${ARCH}\.deb")

for KERN_DEB in $KERNELS; do
    VERSION=$(echo $KERN_DEB | \
        grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9]\+" | head -1)

    if echo $TAGS | grep -q "\"${VERSION}\""; then
        echo "${REPO}:${VERSION} exists"
        continue
    fi

    EXTRA_DEB=$(echo $LINKS | \
        grep -o "linux-image-extra-${VERSION}-generic_[^ ]\+${ARCH}\.deb")

    URLS="${BASE_URL}/${KERN_DEB} ${BASE_URL}/${EXTRA_DEB}"

    # Don't pull in the headers. This is mostly for testing
    # HDR_DEB=$(echo $LINKS | \
    #     grep -o "linux-headers-${VERSION}_[^ ]\+_all\.deb")
    # HDR_ARCH_DEB=$(echo $LINKS | \
    #     grep -o "linux-headers-${VERSION}-generic_[^ ]\+_${ARCH}\.deb")
    # URLS="${URLS} ${BASE_URL}/${HDR_DEB} ${BASE_URL}/${HDR_ARCH_DEB}"

    docker build -t ${REPO}:${VERSION} -f Dockerfile.deb --no-cache \
           --build-arg DEB_URLS="${URLS}" . &&
        DOCKER_CONTENT_TRUST=1 docker push ${REPO}:${VERSION}

    docker rmi ${REPO}:${VERSION}
    docker system prune -f
done
