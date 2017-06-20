.PHONY: default all
default: bin/moby bin/linuxkit bin/rtf 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=linuxkit/go-compile:6579a00b44686d0e504d513fc4860094769fe7df

MOBY?=bin/moby
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

MOBY_COMMIT=d8cc1b3f08df02ad563d3f548ac2527931a925a6
bin/moby: Makefile | bin
	docker run --rm --log-driver=none $(CROSS) $(GO_COMPILE) --clone-path github.com/moby/tool --clone https://github.com/moby/tool.git --commit $(MOBY_COMMIT) --package github.com/moby/tool/cmd/moby --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ > tmp_moby_bin.tar
	tar xf tmp_moby_bin.tar > $@
	rm tmp_moby_bin.tar
	touch $@

RTF_COMMIT=34ec986e726d661f2a25ff085d669e057e3e5345
RTF_CMD=github.com/linuxkit/rtf/cmd
bin/rtf: Makefile | bin
	docker run --rm --log-driver=none $(CROSS) $(GO_COMPILE) --clone-path github.com/linuxkit/rtf --clone https://github.com/linuxkit/rtf.git --commit $(RTF_COMMIT) --package github.com/linuxkit/rtf --ldflags "-X $(RTF_CMD).GitCommit=$(RTF_COMMIT) -X $(RTF_CMD).Version=$(VERSION)" -o $@ > tmp_rtf_bin.tar
	tar xf tmp_rtf_bin.tar > $@
	rm tmp_rtf_bin.tar
	touch $@


LINUXKIT_DEPS=$(wildcard src/cmd/linuxkit/*.go) Makefile vendor.conf
bin/linuxkit: $(LINUXKIT_DEPS) | bin
	tar cf - vendor -C src/cmd/linuxkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ > tmp_linuxkit_bin.tar
	tar xf tmp_linuxkit_bin.tar > $@
	rm tmp_linuxkit_bin.tar
	touch $@

local: $(LINUXKIT_DEPS) | bin
	go build -o bin/linuxkit --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" github.com/linuxkit/linuxkit/src/cmd/linuxkit

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
	$(MAKE) GOOS=darwin
	$(MAKE) clean
	$(MAKE) GOOS=linux
	$(MAKE) clean
	$(MAKE) GOOS=windows
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all
	$(MAKE) -C pkg tag

ci-tag:
	$(MAKE) clean
	$(MAKE) GOOS=darwin
	$(MAKE) clean
	$(MAKE) GOOS=linux
	$(MAKE) clean
	$(MAKE) GOOS=windows
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test all
	$(MAKE) -C pkg tag

ci-pr:
	$(MAKE) clean
	$(MAKE) GOOS=darwin
	$(MAKE) clean
	$(MAKE) GOOS=linux
	$(MAKE) clean
	$(MAKE) GOOS=windows
	$(MAKE) clean
	$(MAKE)
	$(MAKE) install
	$(MAKE) -C test pr
	$(MAKE) -C pkg tag

.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *-state *.img *.iso *.gz *.qcow2 *.vhd *.vmx *.vmdk *.tar 
