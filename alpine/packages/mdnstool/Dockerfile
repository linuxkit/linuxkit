FROM golang:alpine

RUN apk update && apk upgrade && apk add git

RUN mkdir -p /go/src/mdnstool
WORKDIR /go/src/mdnstool

COPY . /go/src/mdnstool/
RUN go get
RUN go install
