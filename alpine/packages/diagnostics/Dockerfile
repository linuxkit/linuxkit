FROM mobylinux/alpine-build-go:78d9a136c632c66f0767e11ce3c661989d706f21

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
