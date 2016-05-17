FROM alpine:3.3

RUN apk update && apk upgrade && apk add build-base linux-headers

RUN mkdir -p /llmnrd

COPY src /llmnrd/
RUN tar czvf /llmnrd.tar.gz llmnrd

WORKDIR /llmnrd
RUN make
