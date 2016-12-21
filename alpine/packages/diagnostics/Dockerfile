# Tag: d3f266a146a79f35d3bebf252cce62eee25fcfa9
FROM mobylinux/alpine-build-go@sha256:1eca9f912cfa4f59ad817acad76744516b999395c080bc01a50b3a2b3a9a3f5c

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN lint.sh .

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
