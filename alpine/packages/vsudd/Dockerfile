FROM mobylinux/alpine-build-go:21b413791b4fed2e16126c77c6b860feaf889d70

COPY ./ /go/src/vsudd/

WORKDIR /go/src/vsudd

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "vsudd"]
