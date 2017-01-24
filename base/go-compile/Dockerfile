FROM golang:1.7-alpine3.5
RUN apk update && apk add --no-cache build-base git

RUN go get -u github.com/golang/lint/golint

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
