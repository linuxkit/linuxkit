FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    musl \
    sfdisk \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin GO111MODULE=off
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY *.go /go/src/mountie/
RUN go-compile.sh /go/src/mountie

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /go/bin/mountie usr/bin/mountie
CMD ["/usr/bin/mountie"]
