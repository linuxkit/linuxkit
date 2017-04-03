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
ENV RUNC_COMMIT=ef9a4b315558d31eae520725ff67383c2f79c3cb
RUN mkdir -p $GOPATH/src/github.com/opencontainers && \
  cd $GOPATH/src/github.com/opencontainers && \
  git clone https://github.com/opencontainers/runc.git
WORKDIR $GOPATH/src/github.com/opencontainers/runc
RUN git checkout $RUNC_COMMIT
# TODO static pie, currently no easy way to change build options
RUN make static BUILDTAGS="seccomp"
RUN cp runc /usr/bin/
WORKDIR /
COPY . .
