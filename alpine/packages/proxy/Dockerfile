FROM golang:alpine

RUN apk update && apk add alpine-sdk

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY ./ /go/src/proxy/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/proxy ] && mv /go/bin/*/proxy /go/bin/ || true
