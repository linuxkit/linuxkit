.PHONY: default all
default: bin/moby 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

MOBY?=bin/moby
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

INFRAKIT_DEPS=$(wildcard src/cmd/infrakit-instance-hyperkit/*.go) Makefile vendor.conf
bin/infrakit-instance-hyperkit: $(INFRAKIT_DEPS) | bin
	docker build --build-arg target=infrakit-instance-hyperkit -t $(GIT_COMMIT)-builder .
	$(eval cid = $(shell docker create $(GIT_COMMIT)-builder))
	docker cp $(cid):/out/moby $@
	docker rm $(cid)

test-initrd.img: $(MOBY) test/test.yml
	bin/moby build test/test.yml

test-bzImage: test-initrd.img

# interactive versions need to use volume mounts
.PHONY: test-qemu-efi
test-qemu-efi: test-efi.iso
	./scripts/qemu.sh $^ 2>&1 | tee test-efi.log
	$(call check_test_log, test-efi.log)

bin:
	mkdir -p $@

install:
	cp -R ./bin/* $(PREFIX)/bin

define check_test_log
	@cat $1 |grep -q 'Kernel config test suite PASSED'
endef

.PHONY: test-hyperkit
test-hyperkit: $(MOBY) test-initrd.img test-bzImage test-cmdline
	rm -f disk.img
	script -q /dev/null $(MOBY) run test | tee test.log
	$(call check_test_log, test.log)

.PHONY: test-gcp
test-gcp: $(MOBY) test.img.tar.gz
	script -q /dev/null $(MOBY) run gcp test.img.tar.gz | tee test-gcp.log
	$(call check_test_log, test-gcp.log)

.PHONY: test
test: test-initrd.img test-bzImage test-cmdline
	tar cf - $^ | ./scripts/qemu.sh 2>&1 | tee test.log
	$(call check_test_log, test.log)

.PHONY: ci ci-tag ci-pr
ci:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

ci-tag:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

ci-pr:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

.PHONY: clean
clean:
	rm -rf bin *.log *-bzImage *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd
