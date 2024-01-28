FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build

RUN apk add --no-cache go musl-dev
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"
ENV GOPATH=/go PATH=$PATH:/go/bin

# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY . /go/src/logwrite/
RUN go-compile.sh /go/src/logwrite

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/logwrite usr/bin/logwrite
CMD ["/usr/bin/logwrite"]
