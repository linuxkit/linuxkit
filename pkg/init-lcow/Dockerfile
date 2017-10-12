FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    musl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS build
ENV OPENGCS_REPO=https://github.com/Microsoft/opengcs
ENV OPENGCS_COMMIT=48ae4e3ba3d2fea746fb4dc20a72832a46f45466
RUN apk add --no-cache build-base curl git go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
WORKDIR $GOPATH/src/github.com/Microsoft/opengcs
RUN mkdir -p $GOPATH/src/github.com/Microsoft/opengcs && \
    cd $GOPATH/src/github.com/Microsoft/opengcs && \
    git init && \
    git remote add origin $OPENGCS_REPO
RUN git fetch --depth 1 origin $OPENGCS_COMMIT && \
    git checkout $OPENGCS_COMMIT
RUN cd service && \
    make
RUN mkdir /out && \
    cp -r $GOPATH/src/github.com/Microsoft/opengcs/service/bin /out/bin && \
    mkdir /out/sbin && \
    curl -fsSL "https://raw.githubusercontent.com/mirror/busybox/38d966943f5288bb1f2e7219f50a92753c730b14/examples/udhcp/simple.script" -o /out/sbin/udhcpc_config.script && \
    chmod ugo+rx /out/sbin/udhcpc_config.script

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /out/ /
COPY init /
