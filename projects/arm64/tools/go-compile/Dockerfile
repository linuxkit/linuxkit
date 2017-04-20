FROM aarch64/alpine:3.5
RUN apk --no-cache add --update --upgrade --purge   \
  build-base                                        \
  git                                               \
  go
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN go get -u github.com/golang/lint/golint && \
    go get -u github.com/gordonklaus/ineffassign && \
    go get -u github.com/LK4D4/vndr

COPY . ./

ENTRYPOINT ["/compile.sh"]
