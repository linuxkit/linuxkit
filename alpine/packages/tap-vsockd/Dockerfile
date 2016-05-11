FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk util-linux-dev

RUN mkdir -p /tap-vsockd
WORKDIR /tap-vsockd

COPY . /tap-vsockd

RUN make tap-vsockd
