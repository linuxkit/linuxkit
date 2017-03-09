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
ENV RUNC_COMMIT=49a33c41f830c7863c3e53e547d071cd0c0f27b7
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
