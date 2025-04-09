FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    curl \
    git \
    go \
    musl-dev

# curl to install golangci-lint
RUN apk add --no-cache curl

# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache --initdb -p /out gcc || true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

# installing golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /out/go/bin v2.0.2


FROM scratch
ENV GOPATH=/go PATH=$PATH:/go/bin
COPY --from=mirror /out/ /
COPY --from=mirror /go/bin/ /go/bin/
COPY /compile.sh /compile.sh
ENTRYPOINT ["/compile.sh"]
