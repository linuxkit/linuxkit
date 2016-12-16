# Tag: 3da0b0ea0da2724232603094e67c75b41adab551
FROM mobylinux/alpine-build-c@sha256:0236d6599f6c8f7aa42829285e04202fbe99984df2818af0f5a453a59de8b090

ARG KERNEL_VERSION=4.8.14

# get kernel source and extract it under /linux
ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}
RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

RUN mkdir -p /build/perf && \
    make -C /linux/tools/perf O=/build/perf LDFLAGS=-static

WORKDIR /build/perf
CMD ["tar", "cf", "-", "perf"]
    
    
