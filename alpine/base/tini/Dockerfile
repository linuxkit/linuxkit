# Tag: 7b94dce736818ca5c9d5367be360b79714687ca5
FROM mobylinux/alpine-build-c@sha256:9157aa25a781b5c47c563c0b9a5d564d41b32ceac9bf26d1e1249e8849d62c8b

ENV TINI_VERSION=0.13.0
ADD https://github.com/krallin/tini/archive/v${TINI_VERSION}.tar.gz tini-${TINI_VERSION}.tar.gz
RUN zcat tini-${TINI_VERSION}.tar.gz | tar xvf -
WORKDIR tini-${TINI_VERSION}
RUN cmake . && make
RUN cp -a tini-static /bin/tini
