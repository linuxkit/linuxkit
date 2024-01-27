FROM linuxkit/alpine:275ef778d89133ebec5e80d82c664c3549f919d4 AS mirror

RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin GO111MODULE=off
ENV VIRTSOCK_COMMIT=f1e32d3189e0dbb81c0e752a4e214617487eb41f

RUN git clone https://github.com/linuxkit/virtsock.git /go/src/github.com/linuxkit/virtsock && \
    cd /go/src/github.com/linuxkit/virtsock && \
    git checkout $VIRTSOCK_COMMIT && \
    make vsudd

FROM scratch
COPY --from=mirror /go/src/github.com/linuxkit/virtsock/bin/vsudd.linux /vsudd
ENTRYPOINT ["/vsudd"]
