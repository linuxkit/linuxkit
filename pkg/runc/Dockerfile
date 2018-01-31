FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 as alpine
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
ENV RUNC_COMMIT=9f9c96235cc97674e935002fc3d78361b696a69e
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
