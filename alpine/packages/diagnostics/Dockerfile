FROM golang:alpine

RUN apk update && apk add build-base

RUN mkdir -p /go/src/diagnostics
WORKDIR /go/src/diagnostics

COPY . /go/src/diagnostics

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/diagnostics ] && mv /go/bin/*/diagnostics /go/bin/ || true
