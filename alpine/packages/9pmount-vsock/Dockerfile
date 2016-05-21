FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk util-linux-dev linux-headers

RUN mkdir -p /9pmount-vsock
WORKDIR /9pmount-vsock

COPY . /9pmount-vsock

RUN make 9pmount-vsock
