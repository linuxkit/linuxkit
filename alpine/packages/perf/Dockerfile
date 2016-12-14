FROM mobylinux/alpine-build-c@sha256:d66b9625abc831f28f8c584991a9cb6975e85d3bb3d3768474b592f1cf32a3a6

ARG KERNEL_VERSION=4.8.14

# get utilities to build perf utils
RUN \
  apk add \
    binutils-dev \
    bison \
    flex \
    gtk+2.0-dev \
    gtk+-dev \
    libelf-dev \
    perl-dev \
    python-dev \
    slang-dev


# get kernel source and extract it under /linux
ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}
RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

RUN mkdir -p /build/perf && \
    make -C /linux/tools/perf O=/build/perf LDFLAGS=-static

WORKDIR /build/perf
CMD ["tar", "cf", "-", "perf"]
    
    
