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
MOBY_VERSION=0.0
bin/moby: tmp_moby_bin.tar | bin
	tar xf $<
	rm $<
	touch $@

tmp_moby_bin.tar: Makefile
	docker run --rm --log-driver=none $(CROSS) $(GO_COMPILE) --clone-path github.com/moby/tool --clone https://github.com/moby/tool.git --commit $(MOBY_COMMIT) --package github.com/moby/tool/cmd/moby --ldflags "-X main.GitCommit=$(MOBY_COMMIT) -X main.Version=$(MOBY_VERSION)" -o bin/moby > $@

RTF_COMMIT=1268bd2ef979bd840dc35dcb8d5dc0a5c75ba129
RTF_CMD=github.com/linuxkit/rtf/cmd
RTF_VERSION=0.0
bin/rtf: tmp_rtf_bin.tar | bin
	tar xf $<
	rm $<
	touch $@

tmp_rtf_bin.tar: Makefile
	docker run --rm --log-driver=none $(CROSS) $(GO_COMPILE) --clone-path github.com/linuxkit/rtf --clone https://github.com/linuxkit/rtf.git --commit $(RTF_COMMIT) --package github.com/linuxkit/rtf --ldflags "-X $(RTF_CMD).GitCommit=$(RTF_COMMIT) -X $(RTF_CMD).Version=$(RTF_VERSION)" -o bin/rtf > $@


LINUXKIT_DEPS=$(wildcard src/cmd/linuxkit/*.go) Makefile vendor.conf
bin/linuxkit: tmp_linuxkit_bin.tar
	tar xf $<
	rm $<
	touch $@

tmp_linuxkit_bin.tar: $(LINUXKIT_DEPS)
	tar cf - vendor -C src/cmd/linuxkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o bin/linuxkit > $@

# For profiling we add timestamps around some targets
define TIME
@echo "=== [" `date -u -R` "]"
endef

.PHONY: test-cross
test-cross:
	$(TIME)
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=darwin tmp_moby_bin.tar tmp_rtf_bin.tar tmp_linuxkit_bin.tar
	$(TIME)
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=windows tmp_moby_bin.tar tmp_rtf_bin.tar tmp_linuxkit_bin.tar
	$(TIME)
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=linux tmp_moby_bin.tar tmp_rtf_bin.tar tmp_linuxkit_bin.tar
	$(TIME)
	$(MAKE) clean


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
ci: test-cross
	$(MAKE)
	$(TIME)
	$(MAKE) install
	$(TIME)
	$(MAKE) -C test all
	$(TIME)
	$(MAKE) -C pkg tag
	$(TIME)

ci-tag: test-cross
	$(MAKE)
	$(TIME)
	$(MAKE) install
	$(TIME)
	$(MAKE) -C test all
	$(TIME)
	$(MAKE) -C pkg tag
	$(TIME)

ci-pr: test-cross
	$(MAKE)
	$(TIME)
	$(MAKE) install
	$(TIME)
	$(MAKE) -C test pr
	$(TIME)
	$(MAKE) -C pkg tag
	$(TIME)

.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *-state *.img *.iso *.gz *.qcow2 *.vhd *.vmx *.vmdk *.tar 
