.PHONY: default all
default: bin/moby moby-initrd.img
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=mobylinux/go-compile:a2ff853b00d687f845d0f67189fa645a567c006e@sha256:09fff8a5c022fc9ead35b2779209c043196b09193c6e61d98603d402c0971f03

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

moby-initrd.img: bin/moby moby.yml
	bin/moby build moby.yml

moby-bzImage: moby-initrd.img

test-initrd.img: bin/moby test/test.yml
	bin/moby build test/test.yml

test-bzImage: test-initrd.img

# interactive versions need to use volume mounts
.PHONY: qemu qemu-iso qemu-efi test-qemu-efi
qemu: moby-initrd.img moby-bzImage moby-cmdline
	./scripts/qemu.sh moby-initrd.img moby-bzImage "$(shell cat moby-cmdline)"

qemu-iso: alpine/mobylinux-bios.iso
	./scripts/qemu.sh $^

qemu-efi: moby-efi.iso
	./scripts/qemu.sh $^

test-qemu-efi: test-efi.iso
	./scripts/qemu.sh $^ 2>&1 | tee test-efi.log
	$(call check_test_log, test-efi.log)

bin:
	mkdir -p $@

.PHONY: hyperkit
hyperkit: bin/moby moby-initrd.img moby-bzImage moby.yml
	bin/moby run moby

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
endef

.PHONY: hyperkit-test
hyperkit-test: bin/moby test-initrd.img test-bzImage test-cmdline
	rm -f disk.img
	script -q /dev/null bin/moby run test | tee test.log
	$(call check_test_log, test.log)

.PHONY: test
test: test-initrd.img test-bzImage test-cmdline
	tar cf - $^ | ./scripts/qemu.sh 2>&1 | tee test.log
	$(call check_test_log, test.log)

.PHONY: ebpf
EBPF_TAG=ebpf/ebpf.tag
EBPF_IMAGE=mobylinux/ebpf:$(MEDIA_PREFIX)$(TAG)
ebpf: alpine/initrd.img kernel/x86_64/vmlinuz64
ifeq ($(STATUS),)
	[ -f $(EBPF_TAG) ]
	docker tag $(shell cat $(EBPF_TAG)) $(EBPF_IMAGE)
	docker push $(EBPF_IMAGE)
else
	$(error "git not clean")
endif

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
