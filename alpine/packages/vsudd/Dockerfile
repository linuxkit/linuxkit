FROM mobylinux/alpine-build-go:30067067003d565887d7efe533eba03ed46038d2

RUN mkdir -p /go/src/vsudd
WORKDIR /go/src/vsudd

COPY . /go/src/vsudd/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/vsudd ] && mv /go/bin/*/vsudd /go/bin/ || true
