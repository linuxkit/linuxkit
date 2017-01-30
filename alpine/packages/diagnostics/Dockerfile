# Tag: d7262323ec56ef884f04a58447e2417364680544
FROM mobylinux/alpine-build-go@sha256:dda58e831bf72d6b59928d963f7dae65ed0d9d52398e99be9c2456c2da66a7fd

COPY ./ /go/src/diagnostics-server/

WORKDIR /go/src/diagnostics-server

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "diagnostics-server"]
