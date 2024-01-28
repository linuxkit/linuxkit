# Dockerfile to build linuxkit/init for linuxkit
FROM linuxkit/containerd-dev:25522a7fcffd14465d807fadc3d3e4f6da7b10ec AS containerd-dev
FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev git linux-headers

ADD usermode-helper.c ./
RUN LDFLAGS=-static CFLAGS=-Werror make usermode-helper

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"

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

FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl

# Add /etc/ssl/certs so it can be bind-mounted into metadata package
RUN mkdir -p /out/etc/ssl/certs

# Remove cache residuals. We retain apk for SBOM tools
RUN rm -rf /out/var/cache

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
