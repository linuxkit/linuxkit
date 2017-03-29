# Tag: cc5bf4d39442741eaeffad79bbd9ae049a9253fa
FROM mobylinux/alpine-build-ebpf@sha256:da37d99a8baad18e68c44d3807c86977767ed90ce03133b508842eea3b786d3a

ADD kernel-headers.tar /
ADD kernel-dev.tar /
ADD kernel-modules.tar /
RUN cd elfutils-$ELFUTILS_VERSION && \
  automake && \
  ./configure --prefix=/usr CFLAGS=-Wno-strict-aliasing && \
  make -C libelf && make -C libelf install
RUN mkdir -p bcc/build && cd bcc/build && \
  cmake .. -DCMAKE_INSTALL_PREFIX=/usr -DLUAJIT_INCLUDE_DIR=/usr/include/luajit-2.1 && \
  make && \
  make install
RUN mkdir -p /usr/local/share/lua/5.1/ && cd ljsyscall && cp -a *.lua syscall /usr/local/share/lua/5.1/

ENV LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:/usr/lib64
