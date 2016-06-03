FROM alpine:3.4

RUN apk update && apk upgrade && apk add build-base

RUN mkdir -p /transfused
WORKDIR /transfused

COPY . /transfused

RUN make transfused
