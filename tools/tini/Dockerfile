FROM linuxkit/c-compile:f52f485825c890d581e82a62af6906c1d33d8e5d@sha256:473fd283a090d3546e93915adacf00c69a23ff615c95c968c4a40e8723985853 AS tini-build
COPY . /
ENV TINI_VERSION=0.13.0
ADD https://github.com/krallin/tini/archive/v${TINI_VERSION}.tar.gz tini-${TINI_VERSION}.tar.gz

RUN zcat tini-${TINI_VERSION}.tar.gz | tar xvf -
RUN cd tini-${TINI_VERSION} && cmake . && make && cp -a tini-static /bin/tini

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=tini-build bin/tini bin/tini
