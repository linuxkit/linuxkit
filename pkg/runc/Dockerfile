FROM golang:1.7-alpine3.5
RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  bash \
  gcc \
  git \
  libc-dev \
  libseccomp-dev \
  linux-headers \
  make \
  && true
ENV RUNC_COMMIT=ac50e77bbb440dcab354a328c79754e2502b79ca
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
# TODO static pie, currently no easy way to change build options
RUN make static BUILDTAGS="seccomp"
RUN cp runc /usr/bin/
WORKDIR /
RUN printf "FROM scratch\nCOPY /usr/bin/runc /usr/bin/\n" > Dockerfile
CMD ["tar", "cf", "-", "Dockerfile", "usr/bin/runc"]
