FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin GO111MODULE=off
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY logwrite.go /go/src/logwrite/
RUN go-compile.sh /go/src/logwrite

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/logwrite usr/bin/logwrite
CMD ["/usr/bin/logwrite"]
