#! /bin/sh

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <org/repo> <ABI version> <kernel version>"
    echo
    echo "Example:"
    echo "$0 foobar/kernel-debian 4.14.0-2 4.14.7-1"
    echo
    echo "This will create a local LinuxKit kernel package:"
    echo "foobar/kernel-debian:4.14.7-1"
    echo "which you can then push to hub or just use locally"
    exit 1
fi

# List all available kernels with:
# curl -s http://mirrors.kernel.org/debian/pool/main/l/linux/ | sed -n 's/.*href="\([^"]*\).*/\1/p' | grep -o "linux-image-[0-9]\.[0-9]\+\.[0-9]\+-[0-9]\+-amd64[^ ]\+_amd64\.deb

REPO=$1
VER1=$2
VER2=$3
URL=http://mirrors.kernel.org/debian/pool/main/l/linux
ARCH=amd64

KERNEL_DEB="${URL}/linux-image-${VER1}-${ARCH}_${VER2}_${ARCH}.deb"
HEADERS_DEB="${URL}/linux-headers-${VER1}-${ARCH}_${VER2}_${ARCH}.deb"
HEADERS_ALL_DEB="${URL}/linux-headers-${VER1}-all_${VER2}_${ARCH}.deb"

DEB_URLS="${KERNEL_DEB} ${HEADERS_DEB} ${HEADERS_ALL_DEB}"

docker build -t "${REPO}:${VER2}" -f Dockerfile.deb --no-cache --build-arg DEB_URLS="${DEB_URLS}" .
