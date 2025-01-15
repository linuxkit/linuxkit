FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 AS mirror

RUN apk add --no-cache go musl-dev
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY . /go/src/sysfs/
RUN go-compile.sh /go/src/sysfs

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/sysfs /usr/bin/sysfs
COPY etc/ /etc/
CMD ["/usr/bin/sysfs"]
