FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk linux-headers

RUN mkdir -p /llmnrd
WORKDIR /llmnrd

COPY src /llmnrd/

RUN make
