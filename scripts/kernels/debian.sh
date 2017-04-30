#! /bin/sh

REPO="linuxkit/kernel-debian"
BASE_URL=http://mirrors.kernel.org/debian/pool/main/l/linux/

TAGS=$(curl --silent -f -lSL https://registry.hub.docker.com/v1/repositories/${REPO}/tags)

ARCH=amd64
LINKS=$(curl -s ${BASE_URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')
# Just get names for 4.x kernels
KERNELS=$(echo $LINKS | \
    grep -o "linux-image-4\.[0-9]\+\.[0-9]\+-[0-9]\+-${ARCH}[^ ]\+_${ARCH}\.deb")

for KERN_DEB in $KERNELS; do
    VERSION=$(echo $KERN_DEB | \
        grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9]\+" | head -1)

    if echo $TAGS | grep -q "\"${VERSION}\""; then
        echo "${REPO}:${VERSION} exists"
        continue
    fi

    URLS="${BASE_URL}/${KERN_DEB}"

    # Doesn't exist build and push
    docker build -t ${REPO}:${VERSION} -f Dockerfile.deb --no-cache \
           --build-arg DEB_URLS="${URLS}" . &&
        DOCKER_CONTENT_TRUST=1 docker push ${REPO}:${VERSION}

    docker rmi ${REPO}:${VERSION}
    docker system prune -f
done
