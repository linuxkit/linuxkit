FROM mobylinux/alpine-build-go:e726e12b5eea95a4e7aa537e416139d52040c68b

COPY ./ /go/src/vsudd/

WORKDIR /go/src/vsudd

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "vsudd"]
