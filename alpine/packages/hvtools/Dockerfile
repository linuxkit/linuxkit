FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk linux-headers

RUN mkdir -p /hvtools
WORKDIR /hvtools

COPY src /hvtools/

RUN make
