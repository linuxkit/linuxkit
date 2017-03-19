.PHONY: default all
default: bin/moby moby-initrd.img
all: default

GO_COMPILE=mobylinux/go-compile:3afebc59c5cde31024493c3f91e6102d584a30b9@sha256:e0786141ea7df8ba5735b63f2a24b4ade9eae5a02b0e04c4fca33b425ec69b0a

MOBY_DEPS=$(wildcard *.go) pkg vendor
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=-e GOOS=$(GOOS) -e GOARCH=$(GOARCH)
endif

bin/moby: $(MOBY_DEPS) | bin
	tar cf - $(MOBY_DEPS) | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby -o $@ | tar xf -

moby-initrd.img: bin/moby moby.yaml
	$^

moby-bzImage: moby-initrd.img

test-initrd.img: bin/moby test.yaml
	$^

test-bzImage: test-initrd.img

# interactive versions need to use volume mounts
.PHONY: qemu qemu-iso
qemu: moby-initrd.img moby-bzImage bin/moby moby.yaml
	./scripts/qemu.sh moby-initrd.img moby-bzImage "$(shell bin/moby --cmdline moby.yaml)"

qemu-iso: alpine/mobylinux-bios.iso
	./scripts/qemu.sh $^

bin:
	mkdir -p $@

DOCKER_HYPERKIT=/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit
DOCKER_VPNKIT=/Applications/Docker.app/Contents/MacOS/vpnkit

bin/com.docker.hyperkit: | bin
ifneq ("$(wildcard $(DOCKER_HYPERKIT))","")
	ln -s $(DOCKER_HYPERKIT) $@
else
	curl -fsSL https://circleci.com/api/v1/project/docker/hyperkit/latest/artifacts/0//Users/distiller/hyperkit/build/com.docker.hyperkit > $@
	chmod a+x $@
endif

bin/vpnkit: | bin
ifneq ("$(wildcard $(DOCKER_VPNKIT))","")
	ln -s $(DOCKER_VPNKIT) $@
else
	curl -fsSL https://circleci.com/api/v1/project/docker/vpnkit/latest/artifacts/0//Users/distiller/vpnkit/vpnkit.tgz \
		| tar xz --strip=2 -C $(dir $@) Contents/MacOS/vpnkit
	touch $@
endif

.PHONY: hyperkit hyperkit-test
hyperkit: scripts/hyperkit.sh bin/com.docker.hyperkit bin/vpnkit moby-initrd.img moby-bzImage moby.yaml
	./scripts/hyperkit.sh moby

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
endef

hyperkit-test: scripts/hyperkit.sh bin/com.docker.hyperkit bin/vpnkit test-initrd.img test-bzImage
	rm -f disk.img
	script -q /dev/null ./scripts/hyperkit.sh test | tee test.log
	$(call check_test_log, test.log)

.PHONY: test
test: test-initrd.img test-bzImage
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
	rm -rf bin disk.img test.log *-initrd.img *-bzImage *.iso
