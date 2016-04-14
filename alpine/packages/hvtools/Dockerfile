FROM alpine:3.3

RUN apk update && apk upgrade && apk add build-base linux-headers

RUN mkdir -p /hvtools

COPY src /hvtools/
RUN tar czvf /hvtools.tar.gz hvtools

WORKDIR /hvtools
RUN make
