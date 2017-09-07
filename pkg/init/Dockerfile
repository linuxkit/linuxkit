FROM linuxkit/alpine:1e67b8fdba849ed2f7e8537c13b7f280639623d6 AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev git linux-headers

ADD usermode-helper.c ./
RUN LDFLAGS=-static CFLAGS=-Werror make usermode-helper

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY cmd /go/src/cmd
RUN go-compile.sh /go/src/cmd/init

RUN cd /go/src/cmd/service && ./skanky-vendor.sh $GOPATH/src/github.com/containerd/containerd
RUN go-compile.sh /go/src/cmd/service

FROM linuxkit/alpine:1e67b8fdba849ed2f7e8537c13b7f280639623d6 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl

# Add /etc/ssl/certs so it can be bind-mounted into metadata package
RUN mkdir -p /out/etc/ssl/certs

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/init /
COPY --from=build /go/bin/service /usr/bin/
COPY --from=build usermode-helper /sbin/
COPY --from=mirror /out/ /
COPY etc etc/
COPY bin bin/
