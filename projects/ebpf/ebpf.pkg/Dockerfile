#FROM mobylinux/alpine-build-ebpf@sha256:da37d99a8baad18e68c44d3807c86977767ed90ce03133b508842eea3b786d3a
ARG IMAGE
FROM ${IMAGE} as ksrc
FROM alpine-build-ebpf:build as build

COPY --from=ksrc /kernel-headers.tar /build
RUN tar xf /build/kernel-headers.tar

COPY --from=ksrc /kernel-dev.tar /build
RUN tar xf /build/kernel-dev.tar

COPY --from=ksrc /kernel.tar /build
RUN tar xf /build/kernel.tar

RUN cd /build
RUN cd elfutils-$ELFUTILS_VERSION && \
  aclocal && \
  automake && \
  ./configure --prefix=/usr CFLAGS=-Wno-strict-aliasing && \
  make -C libelf && make -C libelf install
RUN mkdir -p bcc/build && cd bcc/build && \
  cmake .. -DCMAKE_VERBOSE_MAKEFILE:BOOL=ON -DCMAKE_C_FLAGS="-I/build/usr/include" -DCMAKE_CXX_FLAGS="-I/build/usr/include" -DCMAKE_INSTALL_PREFIX=/usr -DLUAJIT_INCLUDE_DIR=/usr/include/luajit-2.1 && \
  make && \
  make install
RUN mkdir -p /usr/local/share/lua/5.1/ && cd ljsyscall && cp -a *.lua syscall /usr/local/share/lua/5.1/

ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:/usr/lib64

FROM alpine:3.7 as mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk update && apk upgrade -a && \
  apk add --no-cache --initdb -p /out \
  busybox \
  luajit \
  python \
  strace \
  zlib \
  && true

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out /
COPY --from=build /build/usr/src /usr/src/
COPY --from=build /build/usr/include /usr/include/
COPY --from=build /usr/lib/libelf* /usr/lib/
COPY --from=build /usr/lib/libstdc* /usr/lib/
COPY --from=build /usr/lib/libintl* /usr/lib/
COPY --from=build /usr/lib64 /usr/lib/
COPY --from=build /usr/lib/python2.7/site-packages /usr/lib/python2.7/site-packages/
COPY --from=build /usr/share/bcc /usr/share/bcc/
COPY --from=build /usr/bin/bcc-lua /usr/bin/
COPY --from=build /usr/local/share/lua /usr/local/share/lua/
