FROM linuxkit/alpine:c5021a113803f7608e3908014d316c3490183d0b@sha256:4c7a80173c71ca5019dc56f40442b3a8345f141dd46593bd1fe6d130294d07cb AS build
RUN apk add \
    cmake \
    gcc \
    make \
    musl-dev \
    vim

ENV TINI_VERSION=0.13.0
ADD https://github.com/krallin/tini/archive/v${TINI_VERSION}.tar.gz tini-${TINI_VERSION}.tar.gz

RUN zcat tini-${TINI_VERSION}.tar.gz | tar xvf -
RUN cd tini-${TINI_VERSION} && cmake . && make && cp -a tini-static /bin/tini

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build bin/tini bin/tini
