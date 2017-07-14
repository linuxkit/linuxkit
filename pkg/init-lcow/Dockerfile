FROM linuxkit/alpine:3744607156e6b67e3e7d083b15be9e7722215e73 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    musl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:3744607156e6b67e3e7d083b15be9e7722215e73 AS build
RUN apk add --no-cache build-base curl git go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
ENV OPENGCS_COMMIT=4e4e6f28a03e974e6c32f00a91d51f9605821fdd
RUN git clone https://github.com/Microsoft/opengcs.git /go/src/github.com/Microsoft/opengcs && \
    cd /go/src/github.com/Microsoft/opengcs && \
    git checkout $OPENGCS_COMMIT && \
    cd service && \
    make
RUN mkdir /out && \
    cp -r /go/src/github.com/Microsoft/opengcs/service/bin /out/bin && \
    cp /go/src/github.com/Microsoft/opengcs/kernelconfig/4.11/scripts/init_script /out/init && \
    chmod ugo+rx /out/init && \
    mkdir /out/sbin && \
    curl -fSL "https://raw.githubusercontent.com/mirror/busybox/38d966943f5288bb1f2e7219f50a92753c730b14/examples/udhcp/simple.script" -o /out/sbin/udhcpc_config.script && \
    chmod ugo+rx /out/sbin/udhcpc_config.script && \
    mkdir -p /out/root/integration && \
    cp /go/src/github.com/Microsoft/opengcs/kernelconfig/4.11/prebuildSandbox.vhdx /out/root/integration/prebuildSandbox.vhdx

# This line and the one below should be removed once
# https://github.com/Microsoft/opengcs/issues/52 is addressed and then
# runc should be added via the YAML file
FROM linuxkit/runc:4a35484aa6f90a1f06cdf1fb36f7056926a084b9 AS runc

# This is also a hack. opengcs currently hardcodes udhcpc which we
# don't have in alpine. So we use dhcpcd and a hacky script to emulate
# the one command it invokes.
FROM linuxkit/dhcpcd:4b7b8bb024cebb1bbb9c8026d44d7cbc8e202c41 AS dhcpd
RUN mkdir -p /out/sbin && cp /sbin/dhcpcd /out/sbin/dhcpcd && \
    mkdir -p /out/etc && cp /dhcpcd.conf /out/etc/dhcpcd.conf && \
    mkdir -p /out/usr/lib && cp -r /usr/lib/dhcpcd /out/usr/lib/dhcpcd

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /out/ /
COPY --from=runc /usr/bin/runc /sbin/runc
COPY --from=dhcpd /out/ /
COPY /dhcpd-hack.sh /bin/udhcpc
