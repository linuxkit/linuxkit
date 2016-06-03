FROM alpine:3.4

RUN apk update && apk upgrade && apk add build-base util-linux-dev

RUN mkdir -p /nc-vsock
WORKDIR /nc-vsock

COPY . /nc-vsock

RUN make nc-vsock
