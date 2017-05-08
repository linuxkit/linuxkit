FROM linuxkit/alpine:d0cef04aa75159e373fa08a49478ed6bf4adb9b4@sha256:4d8e181db968645b8b3308d2fe725e6f7bb9d9d44a9c3c7782e86c02a6d9e0f1 as alpine
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
ENV GOPATH=/root/go
ENV RUNC_COMMIT=ac50e77bbb440dcab354a328c79754e2502b79ca
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
# TODO static pie, currently no easy way to change build options
RUN make static BUILDTAGS="seccomp"
RUN cp runc /usr/bin/

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /usr/bin/runc /usr/bin/
