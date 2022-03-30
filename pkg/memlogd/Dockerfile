FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY . /go/src/memlogd
RUN go-compile.sh /go/src/memlogd/cmd/memlogd
RUN go-compile.sh /go/src/memlogd/cmd/logread
RUN go-compile.sh /go/src/memlogd/cmd/logwrite

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/memlogd usr/bin/memlogd
COPY --from=build /go/bin/logread usr/bin/logread
COPY --from=build /go/bin/logwrite usr/bin/logwrite
# We'll start from init.d
COPY etc/ /etc/
