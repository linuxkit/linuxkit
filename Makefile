VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

default: moby

DEPS=$(wildcard cmd/moby/*.go src/moby/*.go src/initrd/*.go src/pad4/*.go) vendor.conf Makefile
PREFIX?=/usr/local

GOLINT:=$(shell command -v golint 2> /dev/null)
INEFFASSIGN:=$(shell command -v ineffassign 2> /dev/null)

moby: $(DEPS)
	go build --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ github.com/moby/tool/cmd/moby

.PHONY:
lint:
ifndef GOLINT
	$(error "Please install golint! go get -u github.com/tool/lint")
endif
ifndef INEFFASSIGN
	$(error "Please install ineffassign! go get -u github.com/gordonklaus/ineffassign")
endif
	# golint
	@test -z "$(shell find . -type f -name "*.go" -not -path "./vendor/*" -not -name "*.pb.*" -exec golint {} \; | tee /dev/stderr)"
	# gofmt
	@test -z "$$(gofmt -s -l .| grep -v .pb. | grep -v vendor/ | tee /dev/stderr)"
	# ineffassign
	@test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec ineffassign {} \; | tee /dev/stderr)
ifeq ($(GOOS),)
	# govet
	@test -z "$$(go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr)"
endif

test: lint moby
	# go test
	@go test github.com/moby/tool/src/moby
	# test build
	./moby build -format tar test/test.yml
	rm moby test.tar

.PHONY: install
install: moby
	cp -a $^ $(PREFIX)/bin/

.PHONY: clean
clean:
	rm -f moby
