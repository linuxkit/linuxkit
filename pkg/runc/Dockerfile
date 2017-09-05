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
ENV RUNC_COMMIT=2e7cfe036e2c6dc51ccca6eb7fa3ee6b63976dcd
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN mkdir -p $GOPATH/src/github.com/opencontainers/runc && \
  cd $GOPATH/src/github.com/opencontainers/runc && \
  git init && \
  git remote add origin https://github.com/opencontainers/runc.git
RUN git fetch --depth 1 origin $RUNC_COMMIT && \
  git checkout $RUNC_COMMIT
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
