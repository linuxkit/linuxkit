FROM golang:alpine

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY . /go/src/proxy/

ARG GOARCH
ARG GOOS

RUN go install

RUN [ -f /go/bin/*/proxy ] && mv /go/bin/*/proxy /go/bin/ || true
