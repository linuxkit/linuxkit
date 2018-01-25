FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    iproute2 \
    jq \
    musl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS build
RUN apk add --no-cache \
    build-base \
    git \
    go \
    musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
ENV VIRTSOCK_COMMIT=0416e3d85541e7a067fd000c69f50997b5d47c93
RUN git clone https://github.com/linuxkit/virtsock.git /go/src/github.com/linuxkit/virtsock && \
    cd /go/src/github.com/linuxkit/virtsock && \
    git checkout $VIRTSOCK_COMMIT && \
    make bin/sock_stress.linux


FROM scratch
COPY --from=mirror /out/ /
COPY --from=build /go/src/github.com/linuxkit/virtsock/bin/sock_stress.linux /rootfs/sock_stress
COPY config.template.json *.sh /

LABEL org.mobyproject.config='{"pid": "host", "net":"host", "binds": ["/dev:/dev","/sys:/sys", "/usr/bin/runc:/usr/bin/runc"], "capabilities": ["all"]}'
