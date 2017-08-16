FROM linuxkit/alpine:0fd732eb9e99c4db0953ae8de23d95de340ab847 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    musl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS build
ENV OPENGCS_REPO=https://github.com/Microsoft/opengcs
ENV OPENGCS_COMMIT=2dad06f038530dc305fb139a281507f1002e7e51
RUN apk add --no-cache build-base curl git go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN git clone $OPENGCS_REPO /go/src/github.com/Microsoft/opengcs && \
    cd /go/src/github.com/Microsoft/opengcs && \
    git checkout $OPENGCS_COMMIT && \
    cd service && \
    make
RUN mkdir /out && \
    cp -r /go/src/github.com/Microsoft/opengcs/service/bin /out/bin && \
    mkdir /out/sbin && \
    curl -fSL "https://raw.githubusercontent.com/mirror/busybox/38d966943f5288bb1f2e7219f50a92753c730b14/examples/udhcp/simple.script" -o /out/sbin/udhcpc_config.script && \
    chmod ugo+rx /out/sbin/udhcpc_config.script

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /out/ /
COPY init /
