FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf AS mirror

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
