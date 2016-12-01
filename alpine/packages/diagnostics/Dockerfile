# Tag: 78d9a136c632c66f0767e11ce3c661989d706f21
FROM mobylinux/alpine-build-go@sha256:d82b855d16994ae94f10fa45d0c31e99663aa3e8e4254ef5f31129b899de8ce0

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
