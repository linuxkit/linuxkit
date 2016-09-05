FROM mobylinux/alpine-build-go:f87b7d1c1cdec779ed602bfa5eaaeb94896d612c

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY ./ /go/src/proxy/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/proxy ] && mv /go/bin/*/proxy /go/bin/ || true
