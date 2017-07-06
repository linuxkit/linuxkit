FROM linuxkit/alpine:8bb8664eec04e02a8a131c53aa7d5d94119270ef as alpine
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
ENV RUNC_COMMIT=429a5387123625040bacfbb60d96b1cbd02293ab
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
# TODO static pie, currently no easy way to change build options
RUN make static BUILDTAGS="seccomp"
RUN cp runc /usr/bin/

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=alpine /usr/bin/runc /usr/bin/
