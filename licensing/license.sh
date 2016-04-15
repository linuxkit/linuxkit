#!/bin/bash

fetch () {
  wget $1
  [ $? == 0 ] && exit 0
  # try at archive if original source fails
  BASE=$(basename $1)
  # distfiles are split as v3.3, not v3.3.3
  # edge is not available but should always be upstream
  ALPINE=$(cat /hostetc/alpine-release | sed 's/^\([0-9][0-9]*\.[0-9][0-9]*\).*$/\1/')
  wget http://distfiles.alpinelinux.org/distfiles/v${ALPINE}/$BASE
  [ $? == 0 ] && exit 0
  printf "\e[31m$1 \e[0m\n"
  exit 1
}

cat /hostetc/issue | grep -q Moby || ( printf "You must run this script with -v /etc:/hostetc -v /lib:/lib\n" && exit 1 )

apk info | grep -q fuse || ( printf "You must run this script with -v /etc:/etc -v /lib:/lib\n" && exit 1 )

[ -f /hostetc/kernel-source-info ] || ( printf "Missing kernel source version info\n" && exit 1 )

. /hostetc/kernel-source-info

rm -rf /output/*

cd /aports
git pull

packages.lua | while read l
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
    if [ ! -d "$srcdir"/$pkgname-$pkgver ]
    then
      mkdir -p "$srcdir"/$pkgname-$pkgver
      while read ff
      do
	for f in $ff
	do
	  if [ -n "$(echo $f | tr -d '[[:space:]]')" ]
	    then
              f=$(echo $f | sed 's/^.*:://')
	      printf "looking for source for: $f\n"
              if [ -f "$f" ]
              then
                cp -a $f "$srcdir"/$pkgname-$pkgver/
              else
                ( cd "$srcdir"/$pkgname-$pkgver && \
                fetch $f )
              fi
	  fi
        done
      done <<< "$source"
    fi
  )
done

mkdir -p /output/kernel
cd /output/kernel
cp /proc/config.gz .
wget ${KERNEL_SOURCE=} || ( printf "Failed to download kernel source\n" && exit 1 )
cp -r /hostetc/kernel-patches /output/kernel/patches

git clone -b "$AUFS_BRANCH" "$AUFS_REPO" /output/kernel/aufs
cd /output/kernel/aufs
git checkout -q "$AUFS_COMMIT"
# to make it easier to check in the output of this script if necessary
rm -rf .git

git clone ${AUFS_TOOLS_REPO} /output/aufs-util
cd /output/aufs-util
git checkout "$AUFS_TOOLS_COMMIT"
rm -rf .git

cp /hostusr/share/src/* /output
cp /hostetc/init.d/chronyd /output

printf "All source code now in output/ directory\n"
