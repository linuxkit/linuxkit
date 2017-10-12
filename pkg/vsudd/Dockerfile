FROM linuxkit/alpine:6211ec0252873c9ff9604101fb256353f5a3832a AS mirror

RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin 
ENV VIRTSOCK_COMMIT=a381dcc5bcddf1d7f449495c373dbf70f8e501c0

RUN git clone https://github.com/linuxkit/virtsock.git /go/src/github.com/linuxkit/virtsock && \
    cd /go/src/github.com/linuxkit/virtsock && \
    git checkout $VIRTSOCK_COMMIT && \
    make vsudd

FROM scratch
COPY --from=mirror /go/src/github.com/linuxkit/virtsock/build/vsudd.linux /vsudd
ENTRYPOINT ["/vsudd"]
