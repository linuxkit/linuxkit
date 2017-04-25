.PHONY: default all
default: bin/moby bin/linuxkit 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

MOBY?=bin/moby
LINUXKIT?=bin/linuxkit
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=--build-arg GOOS=$(GOOS) --build-arg GOARCH=$(GOARCH)
endif
ifeq ($(GOOS),darwin)
default: bin/infrakit-instance-hyperkit
endif

PREFIX?=/usr/local/

MOBY_DEPS=$(wildcard src/cmd/moby/*.go) Makefile vendor.conf
MOBY_DEPS+=$(wildcard src/initrd/*.go) $(wildcard src/pad4/*.go)
bin/moby: $(MOBY_DEPS) | bin
	docker build --build-arg ldflags="-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" --build-arg target=moby $(CROSS) -t $(GIT_COMMIT)-builder .
	$(eval cid = $(shell docker create $(GIT_COMMIT)-builder))
	docker cp $(cid):/out/moby $@
	docker rm $(cid)

LINUXKIT_DEPS=$(wildcard src/cmd/linuxkit/*.go) Makefile vendor.conf
bin/linuxkit: $(LINUXKIT_DEPS) | bin
	tar cf - vendor -C src/cmd/linuxkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ > tmp_linuxkit_bin.tar
	tar xf tmp_linuxkit_bin.tar > $@
	rm tmp_linuxkit_bin.tar
	touch $@

INFRAKIT_DEPS=$(wildcard src/cmd/infrakit-instance-hyperkit/*.go) Makefile vendor.conf
bin/infrakit-instance-hyperkit: $(INFRAKIT_DEPS) | bin
	docker build --build-arg target=infrakit-instance-hyperkit -t $(GIT_COMMIT)-builder .
	$(eval cid = $(shell docker create $(GIT_COMMIT)-builder))
	docker cp $(cid):/out/moby $@
	docker rm $(cid)

test-initrd.img: $(MOBY) test/test.yml
	$(MOBY) build test/test.yml

test-bzImage: test-initrd.img

.PHONY: test-qemu-efi
test-qemu-efi: $(LINUXKIT) test-efi.iso
	$(LINUXKIT) run $^ | tee test-efi.log
	$(call check_test_log, test-efi.log)

bin:
	mkdir -p $@

install:
	cp -R ./bin/* $(PREFIX)/bin

define check_test_log
	@cat $1 |grep -q 'test suite PASSED'
endef

.PHONY: test-hyperkit
test-hyperkit: $(LINUXKIT) test-initrd.img test-bzImage test-cmdline
	rm -f disk.img
	$(LINUXKIT) run test | tee test.log
	$(call check_test_log, test.log)

.PHONY: test-gcp
test-gcp: $(LINUXKIT) test.img.tar.gz
	$(LINUXKIT) push gcp test.img.tar.gz
	$(LINUXKIT) run gcp test | tee test-gcp.log
	$(call check_test_log, test-gcp.log)

.PHONY: test
test: $(LINUXKIT) test-initrd.img test-bzImage test-cmdline
	$(LINUXKIT) run test | tee test.log
	$(call check_test_log, test.log)

test-ltp.img.tar.gz: $(MOBY) test/ltp/test-ltp.yml
	$(MOBY) build test/ltp/test-ltp.yml

.PHONY: test-ltp
test-ltp: $(LINUXKIT) test-ltp.img.tar.gz
	$(LINUXKIT) push gcp test-ltp.img.tar.gz
	$(LINUXKIT) run gcp -skip-cleanup -machine n1-highcpu-4 test-ltp | tee test-ltp.log
	$(call check_test_log, test-ltp.log)

.PHONY: ci ci-tag ci-pr
ci:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test
	$(MAKE) test-ltp

ci-tag:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test
	$(MAKE) test-ltp

ci-pr:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

.PHONY: clean
clean:
	rm -rf bin *.log *-bzImage *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd
