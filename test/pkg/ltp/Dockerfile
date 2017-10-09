# debian:jessie
FROM debian@sha256:476959f29a17423a24a17716e058352ff6fbf13d8389e4a561c8ccc758245937 AS build

ENV LTP_VERSION=20170116
ENV LTP_SOURCE=https://github.com/linux-test-project/ltp/releases/download/${LTP_VERSION}/ltp-full-${LTP_VERSION}.tar.xz
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y curl xz-utils make gcc flex bison automake autoconf

RUN curl -fsSL -o ltp-full-${LTP_VERSION}.tar.xz ${LTP_SOURCE}

RUN cat ltp-full-${LTP_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /ltp-full-${LTP_VERSION} /ltp

RUN cd /ltp \
    && make autotools \
    && ./configure \
    && make -j "$(getconf _NPROCESSORS_ONLN)" all \
    && make install

# debian:jessie-slim
FROM debian@sha256:12d31a3d5a1f7cb272708be35031ba068dec46fa84af6aeb38aef5c8a83e8974
COPY --from=build /opt/ltp/ /opt/ltp/
ADD check.sh ./check.sh
WORKDIR /opt/ltp
ENTRYPOINT ["/bin/sh", "/check.sh"]
LABEL org.mobyproject.config='{"pid": "host", "capabilities": ["all"]}'
