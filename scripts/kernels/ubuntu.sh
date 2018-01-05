#! /bin/sh

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <org/repo> <kernel version> <sub version>"
    echo
    echo "Example:"
    echo "$0 foobar/kernel-ubuntu 4.14.0-13 15"
    echo
    echo "This will create a local LinuxKit kernel package:"
    echo "foobar/kernel-ubuntu:4.14.0-13.15"
    echo "which you can then push to hub or just use locally"
    exit 1
fi

# List all available kernels with:
# curl -s http://mirrors.kernel.org/ubuntu/pool/main/l/linux/ | sed -n 's/.*href="\([^"]*\).*/\1/p' | grep -o "linux-image-[0-9]\.[0-9]\+\.[0-9]\+-[0-9]\+-generic_[^ ]\+amd64\.deb"

REPO=$1
VER1=$2
VER2=$3
URL=http://mirrors.kernel.org/ubuntu/pool/main/l/linux
ARCH=amd64

KERNEL_DEB="${URL}/linux-image-${VER1}-generic_${VER1}.${VER2}_${ARCH}.deb"
KERNEL_EXTRA_DEB="${URL}/linux-image-extra-${VER1}-generic_${VER1}.${VER2}_${ARCH}.deb"
HEADERS_DEB="${URL}/linux-headers-${VER1}-generic_${VER1}.${VER2}_${ARCH}.deb"
HEADERS_ALL_DEB="${URL}/linux-headers-${VER1}_${VER1}.${VER2}_all.deb"

DEB_URLS="${KERNEL_DEB} ${KERNEL_EXTRA_DEB} ${HEADERS_DEB} ${HEADERS_ALL_DEB}"

docker build -t "${REPO}:${VER1}.${VER2}" -f Dockerfile.deb --no-cache --build-arg DEB_URLS="${DEB_URLS}" .
