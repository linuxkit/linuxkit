FROM golang:alpine

RUN apk update && apk upgrade && apk add git

RUN mkdir -p /go/src/9pudc
WORKDIR /go/src/9pudc

COPY . /go/src/9pudc/

ARG GOARCH
ARG GOOS

RUN go get && go install

RUN [ -f /go/bin/*/9pudc ] && mv /go/bin/*/9pudc /go/bin/ || true
