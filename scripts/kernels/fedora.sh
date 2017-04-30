#! /bin/sh

REPO="linuxkit/kernel-fedora"
BASE_URL=http://mirrors.kernel.org/fedora/

TAGS=$(curl --silent -f -lSL https://registry.hub.docker.com/v1/repositories/${REPO}/tags)

LINKS=$(curl -s ${BASE_URL}/releases/ | sed -n 's/.*href="\([^"]*\).*/\1/p')
# Just get releases 20+
RELEASES=$(echo $LINKS | grep -o "2[0-9]")

ARCH=x86_64
URLS=""
for RELEASE in $RELEASES; do
    URLS="$URLS ${BASE_URL}/releases/${RELEASE}/Everything/${ARCH}/os/Packages/k/"
    URLS="$URLS ${BASE_URL}/updates/${RELEASE}/${ARCH}/k/"
done

for URL in $URLS; do
    PACKAGES=$(curl -s ${URL}/ | sed -n 's/.*href="\([^"]*\).*/\1/p')

    KERNEL_RPMS=$(echo $PACKAGES | \
        grep -o "kernel-[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9]\+\.[^ ]\+\.rpm")
    for KERNEL_RPM in $KERNEL_RPMS; do
        RPM_URLS="${URL}/${KERNEL_RPM}"

        VERSION=$(echo $KERNEL_RPM | \
                      grep -o "[0-9]\+\.[0-9]\+\.[0-9]\+-[0-9\.]\+\.fc[0-9]\+")

        if echo $TAGS | grep -q "\"${VERSION}\""; then
            echo "${REPO}:${VERSION} exists"
            continue
        fi

        CORE_RPM="kernel-core-${VERSION}.${ARCH}.rpm"
        RPM_URLS="${RPM_URLS} ${URL}/${CORE_RPM}"

        MOD_RPM="kernel-modules-${VERSION}.${ARCH}.rpm"
        RPM_URLS="${RPM_URLS} ${URL}/${MOD_RPM}"

        MOD_EXTRA_RPM="kernel-modules-extra-${VERSION}.${ARCH}.rpm"
        RPM_URLS="${RPM_URLS} ${URL}/${MOD_EXTRA_RPM}"

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
