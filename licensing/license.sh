#!/bin/sh
set -x
cat /etc/issue | grep -q Moby || ( printf "You must run this script with -v /etc:/etc -v /lib:/lib\n" && exit 1 )

apk info | grep fuse || ( printf "You must run this script with -v /etc:/etc -v /lib:/lib\n" && exit 1 )

# [ -f /etc/kernel-version-info ] || ( printf "Missing kernel version info\n" && exit 1 )

# . /etc/kernel-version-info

# APORTS=https://github.com/alpinelinux/aports.git
# git clone ${APORTS}

mkdir -p /output

cd /output

gpl.lua | while read l
do
  echo $l
  APORT_PACKAGE=$(echo $l | sed 's/ .*//')
  APORT_COMMIT=$(echo $l | sed 's/^.* //')
  (
    cd /aports
    [ ! -d main/${APORT_PACKAGE} ] && ( printf "Cannot find package ${APORT_PACKAGE} in aports\n" && exit 1 )
    git checkout ${APORT_COMMIT} || ( printf "Cannot find commit ${APORT_COMMIT} for ${APORT_PACKAGE} in aports\n" && exit 1 )
    export srcdir=/output
    cd main/${APORT_PACKAGE}
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
