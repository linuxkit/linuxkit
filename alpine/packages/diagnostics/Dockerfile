# Tag: 2c9434f1c4ff70b102f34a97d2df1a8363a11a65
FROM mobylinux/alpine-build-go@sha256:d528bbf7102e4209bd59ef030d41de9003ab8e42c303956f62b2df47f3e17849

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN lint.sh .

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
