FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk

RUN mkdir -p /transfused
WORKDIR /transfused

COPY . /transfused

RUN make transfused
