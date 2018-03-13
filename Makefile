.DELETE_ON_ERROR:

.PHONY: default all
default: bin/linuxkit bin/rtf
all: default

VERSION="v0.2+"
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=linuxkit/go-compile:7392985c6f55aba61201514174b45ba755fb386e

LINUXKIT?=bin/linuxkit
GOOS?=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH?=amd64
ifneq ($(GOOS),linux)
CROSS+=-e GOOS=$(GOOS)
endif
ifneq ($(GOARCH),amd64)
CROSS+=-e GOARCH=$(GOARCH)
endif

PREFIX?=/usr/local/

RTF_COMMIT=f2409214ca3b719567aa15bb7b363f24876a4d98
RTF_CMD=github.com/linuxkit/rtf/cmd
RTF_VERSION=0.0
bin/rtf: tmp_rtf_bin.tar | bin
	tar xf $<
	rm $<
	touch $@

tmp_rtf_bin.tar: Makefile
	docker run --rm --log-driver=none -e http_proxy=$(http_proxy) -e https_proxy=$(https_proxy) $(CROSS) $(GO_COMPILE) --clone-path github.com/linuxkit/rtf --clone https://github.com/linuxkit/rtf.git --commit $(RTF_COMMIT) --package github.com/linuxkit/rtf --ldflags "-X $(RTF_CMD).GitCommit=$(RTF_COMMIT) -X $(RTF_CMD).Version=$(RTF_VERSION)" -o bin/rtf > $@

# Manifest tool for multi-arch images
MT_COMMIT=bfbd11963b8e0eb5f6e400afaebeaf39820b4e90
MT_REPO=https://github.com/estesp/manifest-tool
bin/manifest-tool: tmp_mt_bin.tar | bin
	tar xf $<
	rm $<
	touch $@

tmp_mt_bin.tar: Makefile
	docker run --rm --log-driver=none -e http_proxy=$(http_proxy) -e https_proxy=$(https_proxy) $(CROSS) $(GO_COMPILE) --clone-path github.com/estesp/manifest-tool --clone $(MT_REPO) --commit $(MT_COMMIT) --package github.com/estesp/manifest-tool --ldflags "-X main.gitCommit=$(MT_COMMIT)" -o bin/manifest-tool > $@

LINUXKIT_DEPS=$(wildcard src/cmd/linuxkit/*.go) $(wildcard src/cmd/linuxkit/*/*.go) Makefile src/cmd/linuxkit/vendor.conf
bin/linuxkit: tmp_linuxkit_bin.tar
	tar xf $<
	rm $<
	touch $@

tmp_linuxkit_bin.tar: $(LINUXKIT_DEPS)
	tar cf - -C src/cmd/linuxkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit/src/cmd/linuxkit --ldflags "-X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.GitCommit=$(GIT_COMMIT) -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.Version=$(VERSION)" -o bin/linuxkit > $@

.PHONY: test-cross
test-cross:
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=darwin tmp_rtf_bin.tar tmp_mt_bin.tar tmp_linuxkit_bin.tar
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=windows tmp_rtf_bin.tar tmp_mt_bin.tar tmp_linuxkit_bin.tar
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=linux tmp_rtf_bin.tar tmp_mt_bin.tar tmp_linuxkit_bin.tar
	$(MAKE) clean

ifeq ($(GOARCH)-$(GOOS),amd64-linux)
LOCAL_BUILDMODE?=pie
endif
LOCAL_BUILDMODE?=default

LOCAL_LDFLAGS += -s -w -extldflags \"-static\" -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.GitCommit=$(GIT_COMMIT) -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.Version=$(VERSION)
LOCAL_TARGET ?= bin/linuxkit

.PHONY: local-check local-build local-test local
local-check: $(LINUXKIT_DEPS)
	@echo gofmt... && o=$$(gofmt -s -l $(filter %.go,$(LINUXKIT_DEPS))) && if [ -n "$$o" ] ; then echo $$o ; exit 1 ; fi
	@echo govet... && go tool vet -printf=false $(filter %.go,$(LINUXKIT_DEPS))
	@echo golint... && set -e ; for i in $(filter %.go,$(LINUXKIT_DEPS)); do golint $$i ; done
	@echo ineffassign... && ineffassign  $(filter %.go,$(LINUXKIT_DEPS))

local-build: $(LINUXKIT_DEPS) | bin
	go build -o $(LOCAL_TARGET) --buildmode $(LOCAL_BUILDMODE) --ldflags "$(LOCAL_LDFLAGS)" github.com/linuxkit/linuxkit/src/cmd/linuxkit

local-test: $(LINUXKIT_DEPS)
	go test $(shell go list github.com/linuxkit/linuxkit/src/cmd/linuxkit/... | grep -v ^github.com/linuxkit/linuxkit/src/cmd/linuxkit/vendor/)

local: local-check local-build local-test

bin:
	mkdir -p $@

install:
	cp -R ./bin/* $(PREFIX)/bin

.PHONY: test
test:
	$(MAKE) -C test

.PHONY: collect-artifacts
collect-artifacts: artifacts/test.img.tar.gz artifacts/test-ltp.img.tar.gz

.PHONY: ci ci-tag ci-pr
ci: test-cross
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all
	$(MAKE) -C pkg build

ci-tag: test-cross
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all
	$(MAKE) -C pkg build

ci-pr: test-cross
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test pr
	$(MAKE) -C pkg build

.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *-state *.img *.iso *.gz *.qcow2 *.vhd *.vmx *.vmdk *.tar *.raw
