#! /bin/sh

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <org/repo> <base url> <kernel version>"
    echo
    echo "Example:"
    echo "$0 foobar/kernel-fedora http://mirrors.kernel.org/fedora/releases/27/Everything/x86_64/os/Packages/k/ 4.13.9-300.fc27"
    echo
    echo "This will create a local LinuxKit kernel package:"
    echo "foobar/kernel-fedora:4.13.9-300.fc27"
    echo "which you can then push to hub or just use locally"
    exit 1
fi

REPO=$1
URL=$2
VER=$3
ARCH=x86_64

KERNEL_RPM="$URL/kernel-$VER.$ARCH.rpm"
CORE_RPM="$URL/kernel-core-$VER.$ARCH.rpm"
MOD_RPM="$URL/kernel-modules-$VER.$ARCH.rpm"
MOD_EXTRA_RPM="$URL/kernel-modules-extra-$VER.$ARCH.rpm"
HEADERS_RPM="$URL/kernel-headers-$VER.$ARCH.rpm"

RPM_URLS="$KERNEL_RPM $CORE_RPM $MOD_RPM $MOD_EXTRA_RPM $HEADERS_RPM"

docker build -t "$REPO:$VER" -f Dockerfile.rpm --no-cache --build-arg RPM_URLS="$RPM_URLS" .
