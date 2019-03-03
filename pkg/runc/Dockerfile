FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed as alpine
RUN \
  apk add \
  bash \
  gcc \
  git \
  go \
  libc-dev \
  libseccomp-dev \
  linux-headers \
  make \
  && true
ENV GOPATH=/go PATH=$PATH:/go/bin
ENV RUNC_COMMIT=6635b4f0c6af3810594d2770f662f34ddc15b40d
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
RUN make static BUILDTAGS="seccomp" EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS="-extldflags \\\"-fno-PIC -static\\\""
RUN cp runc /usr/bin/

RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/010-onboot
RUN mkdir -p /etc/shutdown.d && ln -s /usr/bin/service /etc/shutdown.d/010-onshutdown

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=alpine /usr/bin/runc /usr/bin/
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY --from=alpine /etc/shutdown.d/ /etc/shutdown.d/
