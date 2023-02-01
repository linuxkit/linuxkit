VERSION="v0.8+"

# test suite to run, blank for all
TEST_SUITE ?=
TEST_SHARD ?=

GO_COMPILE=linuxkit/go-compile:c97703655e8510b7257ffc57f25e40337b0f0813

ifeq ($(OS),Windows_NT)
LINUXKIT?=$(CURDIR)/bin/linuxkit.exe
RTF?=bin/rtf.exe
GOOS?=windows
else
LINUXKIT?=$(CURDIR)/bin/linuxkit
RTF?=bin/rtf
GOOS?=$(shell uname -s | tr '[:upper:]' '[:lower:]')
endif
ifneq ($(GOOS),linux)
CROSS+=-e GOOS=$(GOOS)
endif
ifneq ($(GOARCH),amd64)
CROSS+=-e GOARCH=$(GOARCH)
endif

PREFIX?=/usr/local

LOCAL_TARGET?=$(CURDIR)/bin/linuxkit

export VERSION GO_COMPILE GOOS GOARCH LOCAL_TARGET LINUXKIT

.DELETE_ON_ERROR:

.PHONY: default all
default: linuxkit $(RTF)
all: default

RTF_COMMIT=b74a4f7c78e5cddcf7e6d2e6be7be312b9f645fc
RTF_CMD=github.com/linuxkit/rtf/cmd
RTF_VERSION=0.0
$(RTF): tmp_rtf_bin.tar | bin
	tar -C $(dir $(RTF)) -xf $<
	rm $<
	touch $@

tmp_rtf_bin.tar: Makefile
	docker run --rm --log-driver=none -e http_proxy=$(http_proxy) -e https_proxy=$(https_proxy) $(CROSS) $(GO_COMPILE) --clone-path github.com/linuxkit/rtf --clone https://github.com/linuxkit/rtf.git --commit $(RTF_COMMIT) --package github.com/linuxkit/rtf --ldflags "-X $(RTF_CMD).GitCommit=$(RTF_COMMIT) -X $(RTF_CMD).Version=$(RTF_VERSION)" -o $(notdir $(RTF)) > $@

# Manifest tool for multi-arch images
MT_COMMIT=bfbd11963b8e0eb5f6e400afaebeaf39820b4e90
MT_REPO=https://github.com/estesp/manifest-tool
bin/manifest-tool: tmp_mt_bin.tar | bin
	tar xf $<
	rm $<
	touch $@

tmp_mt_bin.tar: Makefile
	docker run --rm --log-driver=none -e http_proxy=$(http_proxy) -e https_proxy=$(https_proxy) $(CROSS) $(GO_COMPILE) --clone-path github.com/estesp/manifest-tool --clone $(MT_REPO) --commit $(MT_COMMIT) --package github.com/estesp/manifest-tool --ldflags "-X main.gitCommit=$(MT_COMMIT)" -o bin/manifest-tool > $@

.PHONY: linuxkit
linuxkit: bin
	make -C ./src/cmd/linuxkit

.PHONY: test-cross
test-cross:
	make -C ./src/cmd/linuxkit test-cross

.PHONY: local local-%
local:
	make -C ./src/cmd/linuxkit local

local-%:
	make -C ./src/cmd/linuxkit $@

bin:
	mkdir -p $@

install:
	cp -R bin/* $(PREFIX)/bin

sign:
	codesign --entitlements linuxkit.entitlements --force -s - $(PREFIX)/bin/linuxkit

.PHONY: test
test:
	$(MAKE) -C test TEST_SUITE=$(TEST_SUITE) TEST_SHARD=$(TEST_SHARD)

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

update-package-tags:
ifneq ($(LK_RELEASE),)
	$(eval tags := $(shell cd pkg; make show-tag | cut -d ':' -f1))
	$(eval image := :$(LK_RELEASE))
else
	$(eval tags := $(shell cd pkg; make show-tag))
	$(eval image := )
endif
	for img in $(tags); do \
		./scripts/update-component-sha.sh --image $${img}$(image); \
	done

.PHONY: build-all-targets
build-all-targets: bin
	$(MAKE) GOOS=darwin GOARCH=arm64 LOCAL_TARGET=$(CURDIR)/bin/linuxkit-darwin-arm64 local-build
	file bin/linuxkit-darwin-arm64
	$(MAKE) GOOS=darwin GOARCH=amd64 LOCAL_TARGET=$(CURDIR)/bin/linuxkit-darwin-amd64 local-build
	file bin/linuxkit-darwin-amd64
	$(MAKE) GOOS=linux GOARCH=arm64 LOCAL_TARGET=$(CURDIR)/bin/linuxkit-linux-arm64 local-build
	file bin/linuxkit-linux-arm64
	$(MAKE) GOOS=linux GOARCH=amd64 LOCAL_TARGET=$(CURDIR)/bin/linuxkit-linux-amd64 local-build
	file bin/linuxkit-linux-amd64
	$(MAKE) GOOS=linux GOARCH=s390x LOCAL_TARGET=$(CURDIR)/bin/linuxkit-linux-s390x local-build
	file bin/linuxkit-linux-s390x
	$(MAKE) GOOS=windows GOARCH=amd64 LOCAL_TARGET=$(CURDIR)/bin/linuxkit-windows-amd64.exe local-build
	file bin/linuxkit-windows-amd64.exe
	cd bin && openssl sha256 -r linuxkit-* | tr -d '*' > checksums.txt
