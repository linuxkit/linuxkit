VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

default: all

DEPS=$(wildcard cmd/moby/*.go src/moby/*.go src/initrd/*.go src/pad4/*.go) vendor.conf Makefile
PREFIX?=/usr/local

GOMETALINTER:=$(shell command -v gometalinter 2> /dev/null)

dist/moby dist/moby-$(GOOS): $(DEPS)
	CGO_ENABLED=0 go build \
		--ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" \
		-o $@ ./cmd/moby

.PHONY: nogopath
nogopath:
	rm -rf src/github.com
	mkdir -p src/github.com/moby
	ln -s $(CURDIR) $(CURDIR)/src/github.com/moby/tool
	cd src/github.com/moby/tool && GOPATH=$(CURDIR) $(MAKE) dist/moby
	rm -rf src/github.com

.PHONY: lint
lint:
ifndef GOMETALINTER
	$(error "Please install gometalinter! go get -u github.com/alecthomas/gometalinter")
endif
	gometalinter --config gometalinter.json ./...

.PHONY: test
test: dist/moby
	@go test $(shell go list ./... | grep -vE '/vendor/')
	# test build
	dist/moby build -format tar test/test.yml
	rm test.tar

.PHONY: all
all: lint test dist/moby

.PHONY: install
install: dist/moby
	cp -a $^ $(PREFIX)/bin/

.PHONY: clean
clean:
	rm -rf dist src/github.com
