FROM mobylinux/alpine-build-go:30067067003d565887d7efe533eba03ed46038d2

RUN mkdir -p /go/src/proxy
WORKDIR /go/src/proxy

COPY ./ /go/src/proxy/

ARG GOARCH
ARG GOOS

RUN go install --ldflags '-extldflags "-fno-PIC"'

RUN [ -f /go/bin/*/proxy ] && mv /go/bin/*/proxy /go/bin/ || true
