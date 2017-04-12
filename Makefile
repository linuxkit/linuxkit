.PHONY: default all
default: bin/moby 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=linuxkit/go-compile:90607983001c2789911afabf420394d51f78ced8@sha256:8b6566c6fd9f3bca31191b919449248d3cb1ca3a562276fca7199e93451d6596

MOBY?=bin/moby
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=-e GOOS=$(GOOS) -e GOARCH=$(GOARCH)
endif
ifeq ($(GOOS),darwin)
default: bin/infrakit-instance-hyperkit
endif

MOBY_DEPS=$(wildcard src/cmd/moby/*.go) Makefile vendor.conf
MOBY_DEPS+=$(wildcard src/initrd/*.go) $(wildcard src/pad4/*.go)
bin/moby: $(MOBY_DEPS) | bin
	tar cf - vendor src/initrd src/pad4 -C src/cmd/moby . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ | tar xf -
	touch $@

INFRAKIT_DEPS=$(wildcard src/cmd/infrakit-instance-hyperkit/*.go) Makefile vendor.conf
bin/infrakit-instance-hyperkit: $(INFRAKIT_DEPS) | bin
	tar cf - vendor -C src/cmd/infrakit-instance-hyperkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby -o $@ | tar xf -
	touch $@

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

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
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
