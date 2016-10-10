FROM mobylinux/alpine-build-go:e3d97551827fd2ea70b8c484615e85986d4e77fc

COPY ./ /go/src/proxy/

WORKDIR /go/src/proxy

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "proxy"]
