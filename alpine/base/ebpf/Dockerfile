# Tag: dd370179c351da043dfef62fcc68f24995e019c8
FROM mobylinux/alpine-build-ebpf@sha256:557bad7c437e8162440cd4d3de0ccd13a871fd78756bd8b50e1c9afcb8d01fc8

COPY error.h /usr/include/
COPY cdefs.h /usr/include/sys/
ADD kernel-headers.tar /
ADD kernel-dev.tar /
ADD kernel-modules.tar /
ADD 100-musl-compat.patch decl.patch intl.patch bcc-gnuism.patch ./
RUN cat elfutils-$ELFUTILS_VERSION.tar.bz2 | tar xjf -
RUN cd elfutils-$ELFUTILS_VERSION && \
  patch -p1 < ../100-musl-compat.patch && \
  patch -p0 < ../decl.patch && \
  patch -p0 < ../intl.patch && \
  automake && \
  ./configure --prefix=/usr CFLAGS=-Wno-strict-aliasing && \
  make -C libelf && make -C libelf install
RUN cd bcc && patch -p0 < ../bcc-gnuism.patch
RUN mkdir -p bcc/build && cd bcc/build && \
  cmake .. -DCMAKE_INSTALL_PREFIX=/usr -DLUAJIT_INCLUDE_DIR=/usr/include/luajit-2.1 && \
  make && \
  make install
RUN mkdir -p /usr/local/share/lua/5.1/ && cd ljsyscall && cp -a *.lua syscall /usr/local/share/lua/5.1/

ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:/usr/lib64
