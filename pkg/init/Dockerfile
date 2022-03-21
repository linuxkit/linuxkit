FROM linuxkit/containerd-dev:e6a8da1e270da1601ed1bb85bb44c4442e5d51be AS containerd-dev
FROM linuxkit/alpine:5d89cd05a567f9bfbe4502be1027a422d46f4a75 AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev git linux-headers

ADD usermode-helper.c ./
RUN LDFLAGS=-static CFLAGS=-Werror make usermode-helper

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY . /go/src
RUN go-compile.sh /go/src/cmd/init
RUN go-compile.sh /go/src/cmd/rc.init
# this makes sure that the multi stage build copies as a symlink
RUN mkdir /tmp/bin && cd /tmp/bin/ && cp /go/bin/rc.init . && ln -s rc.init rc.shutdown

COPY --from=containerd-dev /go/src/github.com/containerd/containerd $GOPATH/src/github.com/containerd/containerd
RUN cd /go/src/cmd/service && ./skanky-vendor.sh $GOPATH/src/github.com/containerd/containerd
RUN go-compile.sh /go/src/cmd/service

FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror
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
COPY --from=build /tmp/bin /bin/
COPY --from=build /go/bin/service /usr/bin/
COPY --from=build usermode-helper /sbin/
COPY --from=mirror /out/ /
COPY etc etc/
