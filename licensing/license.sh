#!/bin/sh

cat /hostetc/issue | grep -q Moby || ( printf "You must run this script with -v /etc:/hostetc -v /lib:/lib\n" && exit 1 )

apk info | grep -q fuse || ( printf "You must run this script with -v /etc:/etc -v /lib:/lib\n" && exit 1 )

[ -f /hostetc/kernel-source-info ] || ( printf "Missing kernel source version info\n" && exit 1 )

. /hostetc/kernel-source-info

rm -rf /output/*

mkdir -p /output/kernel
cd /output/kernel
wget ${KERNEL_SOURCE=} || ( printf "Failed to download kernel source\n" && exit 1 )

# XXX download aufs

# APORTS=https://github.com/alpinelinux/aports.git
# git clone ${APORTS}

mkdir -p /output

cd /output

gpl.lua | while read l
do
  echo $l
  APORT_PACKAGE=$(echo $l | sed 's/ .*//')
  APORT_COMMIT=$(echo $l | sed 's/^.* //')
  APORT_ORIGIN=$(apk search --origin -x -q ${APORT_PACKAGE})
  (
    cd /aports
    [ ! -d main/${APORT_ORIGIN} ] && ( printf "Cannot find package ${APORT_ORIGIN} in aports\n" && exit 1 )
    git checkout ${APORT_COMMIT} || ( printf "Cannot find commit ${APORT_COMMIT} for ${APORT_ORIGIN} in aports\n" && exit 1 )
    export srcdir=/output
    cd main/${APORT_ORIGIN}
    . ./APKBUILD
    mkdir -p "$srcdir"/$pkgname-$pkgver
    for f in $source
    do
      if [ -f $f ]
      then
        cp -a $f "$srcdir"/$pkgname-$pkgver/
      else
        ( cd "$srcdir"/$pkgname-$pkgver && \
        wget $f || ( printf "Cannot retrieve $f\n" && exit )
        )
      fi
    done
  )
done
