#! /bin/sh

if [ "$#" -ne 4 ]; then
    echo "Usage: $0 <org/repo> <base url> <kernel version> <version> <date>"
    echo
    echo "Example:"
    echo "$0 foobar/kernel-mainline v4.14.11 041411 201801022143"
    echo
    echo "This will create a local LinuxKit kernel package:"
    echo "foobar/kernel-mainline:4.14.11"
    echo "which you can then push to hub or just use locally"
    exit 1
fi

REPO=$1
VER=$2
VER1=$3
DATE=$4
BASE_URL=http://kernel.ubuntu.com/~kernel-ppa/mainline
ARCH=amd64
# Strip leading 'v'
KVER=${VER:1}
URL="${BASE_URL}/${VER}"

KERNEL_DEB="${URL}/linux-image-${KVER}-${VER1}-generic_${KVER}-${VER1}.${DATE}_${ARCH}.deb"
HEADERS_DEB="${URL}/linux-headers-${KVER}-${VER1}-generic_${KVER}-${VER1}.${DATE}_${ARCH}.deb"
HEADERS_ALL_DEB="${URL}/linux-headers-${KVER}-${VER1}_${KVER}-${VER1}.${DATE}_all.deb"

DEB_URLS="${KERNEL_DEB} ${HEADERS_DEB} ${HEADERS_ALL_DEB}"

docker build -t "${REPO}:${KVER}" -f Dockerfile.deb --no-cache --build-arg DEB_URLS="${DEB_URLS}" .
