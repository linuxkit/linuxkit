FROM alpine:3.5
RUN apk update && apk add --no-cache build-base git go
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN go get -u github.com/golang/lint/golint

COPY . ./

ENTRYPOINT ["/compile.sh"]
