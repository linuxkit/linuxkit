FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
    curl \
    e2fsprogs \
    jq \
    python \
    py-pip \
    syslinux \
  && true

RUN pip install -U awscli
