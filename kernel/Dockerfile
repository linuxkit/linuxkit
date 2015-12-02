FROM debian:jessie

ENV KERNEL_VERSION 4.1.13

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
  p7zip-full

ADD https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz .

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

COPY kernel_config /linux/.config

RUN jobs=$(nproc); \
    cd /linux && \
    make -j ${jobs} oldconfig && \
    make -j ${jobs} bzImage
