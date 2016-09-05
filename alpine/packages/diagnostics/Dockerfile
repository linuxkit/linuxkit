FROM mobylinux/alpine-build-go:f87b7d1c1cdec779ed602bfa5eaaeb94896d612c

RUN mkdir -p /go/src/diagnostics
WORKDIR /go/src/diagnostics

COPY . /go/src/diagnostics

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/diagnostics ] && mv /go/bin/*/diagnostics /go/bin/ || true
