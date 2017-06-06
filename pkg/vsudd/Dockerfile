FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin 

COPY . /go/src/vsudd/

# We need cgo for linuxkit/virtsock, so not using go-compile.sh
WORKDIR /go/src/vsudd
RUN test -z $(gofmt -s -l .| grep -v .pb. | grep -v vendor/ | tee /dev/stderr) && \
    test -z $(GOOS=linux go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr) && \
    test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec golint {} \; | tee /dev/stderr) && \
    test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec ineffassign {} \; | tee /dev/stderr) && \
    go install -buildmode pie -ldflags "-s -w ${ldflags} -extldflags \"-fno-PIC -static\""

FROM scratch
COPY --from=mirror /go/bin/vsudd /vsudd
ENTRYPOINT ["/vsudd"]
