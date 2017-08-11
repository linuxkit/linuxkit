FROM linuxkit/alpine:6ed3b299f5243acb6459b4993549c5045e4ad7f4 as alpine
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
ENV RUNC_COMMIT=v1.0.0-rc4
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
