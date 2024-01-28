# Dockerfile to build linuxkit/sysctl for linuxkit
FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror

RUN apk add --no-cache go musl-dev
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY . /go/src/sysctl/
RUN go-compile.sh /go/src/sysctl

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/sysctl /usr/bin/sysctl
COPY etc/ /etc/
COPY --from=mirror /etc/apk /etc/apk/
COPY --from=mirror /lib/apk /lib/apk/
CMD ["/usr/bin/sysctl"]
