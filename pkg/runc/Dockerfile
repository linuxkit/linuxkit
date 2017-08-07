FROM linuxkit/alpine:4a9e5f80a774bbea494d49324428a22c6f018865 as alpine
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
ENV RUNC_COMMIT=45bde006ca8c90e089894508708bcf0e2cdf9e13
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
RUN make static BUILDTAGS="seccomp" EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS="-extldflags \\\"-fno-PIC -static\\\""
RUN cp runc /usr/bin/

ADD cmd /go/src/cmd
RUN go-compile.sh /go/src/cmd/onboot
RUN mkdir -p /etc/init.d && ln -s /usr/bin/onboot /etc/init.d/010-onboot
RUN mkdir -p /etc/shutdown.d && ln -s /usr/bin/onboot /etc/shutdown.d/010-onshutdown

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=alpine /usr/bin/runc /go/bin/onboot /usr/bin/
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY --from=alpine /etc/shutdown.d/ /etc/shutdown.d/
