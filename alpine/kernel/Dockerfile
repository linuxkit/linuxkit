FROM ubuntu:15.10

ARG KERNEL_VERSION=4.4.19
ARG ARCH=x86_64
ARG DEBUG=0

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

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
  wget \
  vim \
  gcc-arm-linux-gnueabihf

RUN wget ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

# this is aufs4.4 20160822
ENV AUFS_REPO       https://github.com/sfjro/aufs4-standalone
ENV AUFS_BRANCH     aufs4.4
ENV AUFS_COMMIT     6fadf0831d0bd32d0f33fa345771235e0f4bddfe

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
        /aufs/lockdep-debug.patch \
    ; do \
        patch -p1 < "$patch"; \
    done

COPY kernel_config.x86_64 /linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config.arm /linux/arch/arm/configs/versatile_defconfig
COPY kernel_config.debug /linux/debug_config

RUN if [ $DEBUG -ne "0" ]; then \
    cat /linux/debug_config >> /linux/arch/x86/configs/x86_64_defconfig; \
    cat /linux/debug_config >> /linux/arch/arm/configs/versatile_defconfig; \
    fi

# Apply local patches
COPY patches /patches
RUN cd /linux && \
    mkdir /etc/kernel-patches && \
    set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        cp "$patch" /etc/kernel-patches; \
        patch -p1 < "$patch"; \
    done && \
    cd /etc/kernel-patches && tar cf /kernel-patches.tar .

RUN jobs=$(nproc); \
    cd /linux && \
    make ARCH=$ARCH defconfig && \
    make ARCH=$ARCH oldconfig && \
    make ARCH=$ARCH -j ${jobs} && \
    make ARCH=$ARCH INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    ( cd /tmp/kernel-modules && tar cf /kernel-modules.tar . ) && \
    make ARCH=$ARCH INSTALL_HDR_PATH=/tmp/kernel-headers headers_install

# Build aufs tools, do this here as they need kernel headers and to match aufs
# Fortunately they are built statically linked

# XXX not cross compiled yet!

# aufs-util 20151116
ENV AUFS_TOOLS_REPO https://github.com/Distrotech/aufs-util.git
ENV AUFS_TOOLS_COMMIT b5e7e204036da5e815ddec15a847a03c232771b4

RUN git clone ${AUFS_TOOLS_REPO} && \
    cd /aufs-util && \
    git checkout "$AUFS_TOOLS_COMMIT" && \
    CPPFLAGS="-I/tmp/kernel-headers/include" CFLAGS=$CPPFLAGS LDFLAGS=$CPPFLAGS make && \
    DESTDIR=/tmp/aufs-utils make install && \
    rm -rf /tmp/aufs-utils/usr/lib /tmp/aufs-utils/usr/share && \
    cd /tmp/aufs-utils && tar cf /aufs-utils.tar .

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\nAUFS_REPO=${AUFS_REPO}\nAUFS_BRANCH=${AUFS_BRANCH}\nAUFS_COMMIT=${AUFS_COMMIT}\nAUFS_TOOLS_REPO=${AUFS_TOOLS_REPO}\nAUFS_TOOLS_COMMIT=${AUFS_TOOLS_COMMIT}\n" > /kernel-source-info
