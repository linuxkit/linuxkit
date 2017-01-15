FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add \
  argp-standalone \
  automake \
  bc \
  binutils-dev \
  bison \
  build-base \
  curl \
  flex \
  libelf-dev \
  linux-headers \
  sed \
  tar \
  util-linux-dev \
  xz \
  && true

ARG KERNEL_VERSION=4.9.3

# get kernel source and extract it under /linux
ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}
RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

RUN mkdir -p /build/perf && \
    make -C /linux/tools/perf O=/build/perf LDFLAGS=-static

WORKDIR /build/perf
CMD ["tar", "cf", "-", "perf"]
    
    
