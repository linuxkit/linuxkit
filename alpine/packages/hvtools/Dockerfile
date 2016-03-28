FROM alpine:3.3

RUN apk update && apk upgrade && apk add build-base linux-headers

RUN mkdir -p /hvtools
WORKDIR /hvtools

COPY src /hvtools/

RUN make
