VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

default: moby

DEPS=$(wildcard cmd/moby/*.go) Makefile
PREFIX?=/usr/local

moby: $(DEPS) lint
	go build --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ github.com/moby/tool/cmd/moby

lint:
	@echo "+ $@: golint, gofmt, go vet"
	# golint
	@test -z "$(shell find . -type f -name "*.go" -not -path "./vendor/*" -not -name "*.pb.*" -exec golint {} \; | tee /dev/stderr)"
	# gofmt
	@test -z "$$(gofmt -s -l .| grep -v .pb. | grep -v vendor/ | tee /dev/stderr)"
	# govet
	@test -z "$$(go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr)"

test: moby
	./moby build test/test.yml
	rm moby test.tar

PHONY: install
install: moby
	cp -a $^ $(PREFIX)/bin/
