FROM mobylinux/alpine-build-go:30067067003d565887d7efe533eba03ed46038d2

RUN mkdir -p /go/src/diagnostics
WORKDIR /go/src/diagnostics

COPY . /go/src/diagnostics

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/diagnostics ] && mv /go/bin/*/diagnostics /go/bin/ || true
