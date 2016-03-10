FROM debian:unstable

ARG KERNEL_VERSION=4.1.19
ARG ARCH=x86_64

RUN apt-get update && apt-get -y upgrade && apt-get -y install \
  unzip \
  xz-utils \
  curl \
  bc \
  build-essential \
  cpio \
  gcc libc6 libc6-dev \
  kmod \
  squashfs-tools \
  genisoimage \
  xorriso \
  syslinux \
  isolinux \
  automake \
  pkg-config \
  git \
  ncurses-dev \
  p7zip-full \
  lzop \
  gcc-arm-linux-gnueabihf

ADD https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz .

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

# http://aufs.sourceforge.net/
ENV AUFS_REPO       https://github.com/sfjro/aufs4-standalone
ENV AUFS_BRANCH     aufs4.1.13+
ENV AUFS_COMMIT     9b0fe5a0ac42f9dca6ecf3261178ce101a270948
# we use AUFS_COMMIT to get stronger repeatability guarantees

# Download AUFS and patch kernel
RUN git clone -b "$AUFS_BRANCH" "$AUFS_REPO" /aufs && \
    cd /aufs && \
    git checkout -q "$AUFS_COMMIT" && \
    cd /linux && \
    cp -r /aufs/Documentation /linux && \
    cp -r /aufs/fs /linux && \
    cp -r /aufs/include/uapi/linux/aufs_type.h /linux/include/uapi/linux/ && \
    set -e && for patch in \
        /aufs/aufs*-kbuild.patch \
        /aufs/aufs*-base.patch \
        /aufs/aufs*-mmap.patch \
        /aufs/aufs*-standalone.patch \
        /aufs/aufs*-loopback.patch \
    ; do \
        patch -p1 < "$patch"; \
    done

COPY kernel_config /linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config.arm /linux/arch/arm/configs/versatile_defconfig

RUN jobs=$(nproc); \
    cd /linux && \
    make ARCH=$ARCH defconfig && \
    make ARCH=$ARCH oldconfig && \
    make ARCH=$ARCH -j ${jobs} && \
    make ARCH=$ARCH INSTALL_HDR_PATH=/tmp/kernel-headers headers_install

# Build aufs tools, do this here as they need kernel headers and to match aufs
# Fortunately they are built statically linked

# XXX not cross compiled yet!

ENV AUFS_TOOLS_COMMIT 5e0c348bd8b1898beb1e043b026bcb0e0c7b0d54

RUN git clone https://github.com/Distrotech/aufs-util.git && \
    cd /aufs-util && \
    git checkout "$AUFS_TOOLS_COMMIT" && \
    CPPFLAGS="-I/tmp/kernel-headers/include" CLFAGS=$CPPFLAGS LDFLAGS=$CPPFLAGS make && \
    DESTDIR=/tmp/aufs-utils make install && \
    rm -rf /tmp/aufs-utils/usr/lib /tmp/aufs-utils/usr/share && \
    cd /tmp/aufs-utils && tar cf /aufs-utils.tar .
