FROM golang:alpine

RUN mkdir -p /go/src/mdnstool
WORKDIR /go/src/mdnstool

ARG GOARCH
ARG GOOS

COPY . /go/src/mdnstool/

RUN go install

RUN [ -f /go/bin/*/mdnstool ] && mv /go/bin/*/mdnstool /go/bin/ || true
