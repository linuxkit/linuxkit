VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

default: moby

DEPS=$(wildcard cmd/moby/*.go) Makefile
PREFIX?=/usr/local

moby: $(DEPS)
	go build --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ github.com/moby/tool/cmd/moby

PHONY: install
install: moby
	cp -a $^ $(PREFIX)/bin/
