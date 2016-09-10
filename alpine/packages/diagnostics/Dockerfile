FROM mobylinux/alpine-build-go:30067067003d565887d7efe533eba03ed46038d2

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
