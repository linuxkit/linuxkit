FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror

RUN apk add --no-cache go musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY .  /go/src/kmsg/
RUN go-compile.sh /go/src/kmsg

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/kmsg /usr/bin/kmsg
CMD ["/usr/bin/kmsg"]
