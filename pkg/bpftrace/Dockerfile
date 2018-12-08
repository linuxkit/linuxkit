FROM linuxkit/alpine:ace75d0ec6978762d445083d6c8c6336bdb658ed AS build
RUN apk add --update \
    bison \
    build-base \
    clang-dev \
    clang-static \
    cmake \
    elfutils-dev \
    flex-dev \
    git \
    linux-headers \
    llvm5-dev \
    llvm5-static \
    zlib-dev

# Put LLVM directories where CMake expects them to be
RUN ln -s /usr/lib/cmake/llvm5 /usr/lib/cmake/llvm
RUN ln -s /usr/include/llvm5/llvm /usr/include/llvm
RUN ln -s /usr/include/llvm5/llvm-c /usr/include/llvm-c

ENV GITREPO=https://github.com/iovisor/bpftrace.git
ENV COMMIT=8f7f8214d7dd7bc25b7740a3c0e9a580a89e0244
ENV STATIC_LINKING=on
ENV RUN_TESTS=0

RUN git clone ${GITREPO} /bpftrace \
    && cd /bpftrace/ \ 
    && mkdir -p /bpftrace/build \
    && git checkout ${COMMIT} \
    && chmod +x /bpftrace/docker/build.sh \
    && /bpftrace/docker/build.sh /bpftrace/build Release -j$(nproc)

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /bpftrace/build/src/bpftrace /usr/bin/bpftrace