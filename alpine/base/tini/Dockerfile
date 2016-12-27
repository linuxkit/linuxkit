# Tag: b77cfc4ad0033d4366df830ed697afc7bab458a2
FROM mobylinux/alpine-build-c@sha256:53739ea6042cb0ac39cf6e262012c1c4224206b2c9b719569fe7efa3a381348c

COPY . /
ENV TINI_VERSION=0.13.0
ADD https://github.com/krallin/tini/archive/v${TINI_VERSION}.tar.gz tini-${TINI_VERSION}.tar.gz
RUN zcat tini-${TINI_VERSION}.tar.gz | tar xvf -
WORKDIR tini-${TINI_VERSION}
RUN cmake . && make
RUN cp -a tini-static /bin/tini
