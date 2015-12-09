FROM golang:alpine

RUN apk update && apk upgrade && apk add git

RUN mkdir -p /go/src/9pudc
WORKDIR /go/src/9pudc

COPY . /go/src/9pudc/
RUN go get
RUN go install
