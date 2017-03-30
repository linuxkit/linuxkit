.PHONY: default all
default: bin/moby moby-initrd.img
all: default

GO_COMPILE=mobylinux/go-compile:3afebc59c5cde31024493c3f91e6102d584a30b9@sha256:e0786141ea7df8ba5735b63f2a24b4ade9eae5a02b0e04c4fca33b425ec69b0a

MOBY_DEPS=$(wildcard src/cmd/moby/*.go)
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=-e GOOS=$(GOOS) -e GOARCH=$(GOARCH)
endif
ifeq ($(GOOS),darwin)
default: bin/infrakit-instance-hyperkit
endif

bin/moby: $(MOBY_DEPS) | bin
	tar cf - vendor src/initrd src/pad4 -C src/cmd/moby . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby -o $@ | tar xf -
	touch $@

MOBY_DEPS=$(wildcard src/cmd/infrakit-instance-hyperkit/*.go)
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
	rm -rf bin disk.img test.log *-initrd.img *-bzImage *-cmdline *.iso *.tar.gz *.qcow2 *.vhd
