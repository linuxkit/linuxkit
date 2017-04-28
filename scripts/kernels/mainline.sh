#! /bin/sh

REPO="linuxkit/kernel-mainline"
BASE_URL=http://kernel.ubuntu.com/~kernel-ppa/mainline

build_image() {
    VERSION=$1
    KDIR=$2
    ARCH=amd64

    LINKS=$(curl -s ${BASE_URL}/${KDIR}/ | \
                sed -n 's/.*href="\([^"]*\).*/\1/p')

    IMAGE=$(echo $LINKS | \
            grep -o "linux-image[^ ]\+-generic[^ ]\+${ARCH}.deb" | head -1)
    [ -z "${IMAGE}" ] && return 1
    HDR_GEN=$(echo $LINKS | grep -o "linux-headers[^ ]\+_all.deb" | head -1)
    [ -z "${HDR_GEN}" ] && return 1
    HDR_ARCH=$(echo $LINKS | \
               grep -o "linux-headers[^ ]\+-generic[^ ]\+${ARCH}.deb" | head -1)
    [ -z "${HDR_ARCH}" ] && return 1

    DEB_URL=${BASE_URL}/${KDIR}/${IMAGE}
    HDR_GEN_URL=${BASE_URL}/${KDIR}/${HDR_GEN}
    HDR_ARCH_URL=${BASE_URL}/${KDIR}/${HDR_ARCH}
    echo "Trying to fetch ${VERSION} from ${DEB_URL}"

    docker build -t ${REPO}:${VERSION} -f Dockerfile.deb --no-cache \
           --build-arg DEB_URLS="${DEB_URL} ${HDR_GEN_URL} ${HDR_ARCH_URL}" .
}

LINKS=$(curl -s ${BASE_URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')
# Extract all kernel versions (drop RCs, ckt(?) and other links)
VERSIONS=$(echo $LINKS | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+[^ ]*" | \
           grep -ve '-rc' | grep -ve '-ckt' | uniq)

# Extract 3.16.x and 4.x.x
THREES=$(echo $VERSIONS | grep -o "v3\.16\.[0-9]\+[^ ]*")
FOURS=$(echo $VERSIONS | grep -o "v4\.[0-9]\+\.[0-9]\+[^ ]*")
KDIRS="${THREES} ${FOURS}"

for KDIR in $KDIRS; do
    # Strip the Ubuntu release name for the tag and also the 'v' like with
    # the other kernel packages
    VERSION=$(echo $KDIR | grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+")
    DOCKER_CONTENT_TRUST=1 docker pull ${REPO}:${VERSION} && continue
    build_image ${VERSION} ${KDIR} && \
        DOCKER_CONTENT_TRUST=1 docker push ${REPO}:${VERSION}
done
