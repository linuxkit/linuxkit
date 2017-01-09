FROM alpine:3.5
RUN \
  apk update && apk upgrade && \
  apk add \
  cmake \
  gcc \
  make \
  musl-dev \
  vim \
  && true

COPY . /
ENV TINI_VERSION=0.13.0
ADD https://github.com/krallin/tini/archive/v${TINI_VERSION}.tar.gz tini-${TINI_VERSION}.tar.gz
RUN zcat tini-${TINI_VERSION}.tar.gz | tar xvf -
WORKDIR tini-${TINI_VERSION}
RUN cmake . && make
RUN cp -a tini-static /bin/tini
