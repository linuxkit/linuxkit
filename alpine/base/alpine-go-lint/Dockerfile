FROM golang:1.7-alpine
RUN apk update && apk add --no-cache build-base git

# Get linting tools
RUN go get -u github.com/golang/lint/golint

# Get the linting script and make its source dir
COPY lint.sh .
RUN mkdir /src

CMD ["./lint.sh", "/src"]