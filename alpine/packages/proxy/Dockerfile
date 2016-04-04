FROM golang:alpine

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY * /go/src/proxy/

RUN mkdir -p /go/src/pkg/proxy
COPY pkg/* /go/src/pkg/proxy/
RUN mkdir -p /go/src/vendor/github.com/Sirupsen/logrus
COPY vendor/github.com/Sirupsen/logrus/* /go/src/vendor/github.com/Sirupsen/logrus/

ARG GOARCH
ARG GOOS

RUN go install

RUN [ -f /go/bin/*/proxy ] && mv /go/bin/*/proxy /go/bin/ || true
