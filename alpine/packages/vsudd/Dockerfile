FROM golang:1.7-alpine

RUN apk update && apk add build-base

RUN mkdir -p /go/src/vsudd
WORKDIR /go/src/vsudd

COPY . /go/src/vsudd/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/vsudd ] && mv /go/bin/*/vsudd /go/bin/ || true
