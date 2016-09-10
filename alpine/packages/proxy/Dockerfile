FROM mobylinux/alpine-build-go:30067067003d565887d7efe533eba03ed46038d2

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY ./ /go/src/proxy/

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "proxy"]
