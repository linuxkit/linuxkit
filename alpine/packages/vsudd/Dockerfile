FROM mobylinux/alpine-build-go:f87b7d1c1cdec779ed602bfa5eaaeb94896d612c

RUN mkdir -p /go/src/vsudd
WORKDIR /go/src/vsudd

COPY . /go/src/vsudd/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/vsudd ] && mv /go/bin/*/vsudd /go/bin/ || true
