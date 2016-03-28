FROM golang:alpine

RUN mkdir -p /go/src/hupper
WORKDIR /go/src/hupper

COPY . /go/src/hupper/

ARG GOARCH
ARG GOOS

RUN go install

RUN [ -f /go/bin/*/hupper ] && mv /go/bin/*/hupper /go/bin/ || true
