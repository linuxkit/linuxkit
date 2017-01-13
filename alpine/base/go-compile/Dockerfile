FROM golang:1.7-alpine
RUN apk update && apk add --no-cache build-base git

RUN go get -u github.com/golang/lint/golint

COPY lint.sh compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
