FROM golang:1.8

RUN go get -u github.com/golang/lint/golint && \
	go get -u github.com/gordonklaus/ineffassign && \
	go get -u github.com/LK4D4/vndr

ADD . /go/src/github.com/linuxkit/linuxkit

ARG target=moby
ARG ldflags
ARG GOOS
ARG GOARCH

WORKDIR /go/src/github.com/linuxkit/linuxkit/src/cmd/$target

RUN files=$(find . -type f -name '*.go' -not -path "./vendor/*" -not -name '*.pb.*') && \
	echo "gofmt..." && test -z $(gofmt -s -l $files | tee /dev/stderr) && \
	echo "go vet..." && test -z $(GOOS=linux go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr) && \
	echo "golintr..." && test -z $(golint $files | tee /dev/stderr) && \
	echo "ineffassign..." && test -z $(for file in $files ; do (ineffassign $file | tee /dev/stderr) ; done)

RUN go build -ldflags "${ldflags}" -buildmode pie -o /out/$target .
