#! /bin/sh

REPO="linuxkit/kernel-centos"
BASE_URL=http://mirror.centos.org/centos/

TAGS=$(curl --silent -f -lSL https://registry.hub.docker.com/v1/repositories/${REPO}/tags)

LINKS=$(curl -s ${BASE_URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')
# Just get names for Centos 7
RELEASES=$(echo $LINKS | grep -o "7\.[^ ]*")
RELEASES="7/ $RELEASES"

# Add updates
URLS=""
for RELEASE in $RELEASES; do
    URLS="$URLS ${BASE_URL}/${RELEASE}/os/x86_64/Packages/"
done
URLS="$URLS ${BASE_URL}/7/updates/x86_64/Packages/"

for URL in $URLS; do
    PACKAGES=$(curl -s ${URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')

    KERNEL_RPMS=$(echo $PACKAGES | \
        grep -o "kernel-[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9]\+\.[^ ]\+\.rpm")
    for KERNEL_RPM in $KERNEL_RPMS; do
        RPM_URLS="${URL}/${KERNEL_RPM}"

        VERSION=$(echo $KERNEL_RPM | \
                      grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9\.]\+\.el[0-9]\+")

        if echo $TAGS | grep -q "\"${VERSION}\""; then
            echo "${REPO}:${VERSION} exists"
            continue
        fi

        # Don't pull in the headers. This is mostly for testing
        # HEADERS_RPM="kernel-headers-${VERSION}.x86_64.rpm"
        # RPM_URLS="${RPM_URLS} ${URL}/${HEADERS_RPM}"

        docker build -t ${REPO}:${VERSION} -f Dockerfile.rpm --no-cache \
               --build-arg RPM_URLS="${RPM_URLS}" . &&
            DOCKER_CONTENT_TRUST=1 docker push ${REPO}:${VERSION}

        docker rmi ${REPO}:${VERSION}
        docker system prune -f
    done
done
