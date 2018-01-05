#! /bin/sh

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <org/repo> <base url> <kernel version>"
    echo
    echo "Example:"
    echo "$0 foobar/kernel-centos http://mirror.centos.org/centos/7/os/x86_64/Packages 3.10.0-693.el7"
    echo
    echo "This will create a local LinuxKit kernel package:"
    echo "foobar/kernel-centos:3.10.0-693.el7"
    echo "which you can then push to hub or just use locally"
    exit 1
fi

REPO=$1
URL=$2
VER=$3
ARCH=x86_64

KERNEL_RPM="$URL/kernel-$VER.$ARCH.rpm"
HEADERS_RPM="$URL/kernel-headers-$VER.$ARCH.rpm"

RPM_URLS="$KERNEL_RPM $HEADERS_RPM"

docker build -t "$REPO:$VER" -f Dockerfile.rpm --no-cache --build-arg RPM_URLS="$RPM_URLS" .
