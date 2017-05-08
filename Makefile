.PHONY: default all
default: bin/moby bin/linuxkit 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=linuxkit/go-compile:5bf17af781df44f07906099402680b9a661f999b

MOBY?=bin/moby
LINUXKIT?=bin/linuxkit
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=-e GOOS=$(GOOS) -e GOARCH=$(GOARCH)
endif

PREFIX?=/usr/local/

MOBY_COMMIT=d504afe4795528920ef06af611efd27b74098d5e
bin/moby: | bin
	docker run --rm --log-driver=none $(CROSS) $(GO_COMPILE) --clone-path github.com/moby/tool --clone https://github.com/moby/tool.git --commit $(MOBY_COMMIT) --package github.com/moby/tool/cmd/moby --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ > tmp_moby_bin.tar
	tar xf tmp_moby_bin.tar > $@
	rm tmp_moby_bin.tar
	touch $@

LINUXKIT_DEPS=$(wildcard src/cmd/linuxkit/*.go) Makefile vendor.conf
bin/linuxkit: $(LINUXKIT_DEPS) | bin
	tar cf - vendor -C src/cmd/linuxkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ > tmp_linuxkit_bin.tar
	tar xf tmp_linuxkit_bin.tar > $@
	rm tmp_linuxkit_bin.tar
	touch $@

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
ci:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all

ci-tag:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all

ci-pr:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test pr

.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd *.vmx *.vmdk *.tar
	$(MAKE) -C test clean
