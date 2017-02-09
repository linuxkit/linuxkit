.PHONY: test test-hyperkit qemu qemu-iso qemu-gce media ebpf ci ci-pr get \
        get-regextract test-gce

all:
	$(MAKE) -C alpine

aufs:
	$(MAKE) AUFS=true all

alpine/initrd.img:
	$(MAKE) -C alpine initrd.img

alpine/initrd-test.img:
	$(MAKE) -C alpine initrd-test.img

kernel/x86_64/vmlinuz64:
	$(MAKE) -C kernel

alpine/mobylinux-bios.iso:
	$(MAKE) -C alpine mobylinux-bios.iso

alpine/gce.img.tar.gz:
	$(MAKE) -C alpine gce.img.tar.gz

alpine/gce-test.img.tar.gz:
	$(MAKE) -C alpine gce-test.img.tar.gz

QEMU_IMAGE=mobylinux/qemu:0fb8c648e8ed9ef6b1ec449587aeab6c53872744@sha256:606f30d815102e73bc01c07915dc0d5f153b0252c63f5f0ed1e39621ec656eb5

# interactive versions need to use volume mounts
qemu: alpine/initrd.img kernel/x86_64/vmlinuz64
	docker run -it --rm -v $(CURDIR)/alpine/initrd.img:/tmp/initrd.img -v $(CURDIR)/kernel/x86_64/vmlinuz64:/tmp/vmlinuz64 $(QEMU_IMAGE)

qemu-iso: alpine/mobylinux-bios.iso
	docker run -it --rm -v $(CURDIR)/alpine/mobylinux-bios.iso:/tmp/mobylinux-bios.iso $(QEMU_IMAGE)

qemu-gce: alpine/gce.img.tar.gz
	docker run -it --rm -v $(CURDIR)/alpine/gce.img.tar.gz:/tmp/gce.img.tar.gz $(QEMU_IMAGE)

test-gce: alpine/gce-test.img.tar.gz
	rm -f test.log
	scripts/gce.sh test.log
	$(call check_test_log, test.log)

bin:
	mkdir -p $@

DOCKER_HYPERKIT=/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit
DOCKER_SLIRP=/Applications/Docker.app/Contents/MacOS/com.docker.slirp

bin/com.docker.hyperkit: | bin
ifneq ("$(wildcard $(DOCKER_HYPERKIT))","")
	ln -s $(DOCKER_HYPERKIT) $@
else
	curl -fsSL https://circleci.com/api/v1/project/docker/hyperkit/latest/artifacts/0//Users/distiller/hyperkit/build/com.docker.hyperkit > $@
	chmod a+x $@
endif

bin/com.docker.slirp: | bin
ifneq ("$(wildcard $(DOCKER_SLIRP))","")
	ln -s $(DOCKER_SLIRP) $@
else
	curl -fsSL https://circleci.com/api/v1/project/docker/vpnkit/latest/artifacts/0//Users/distiller/vpnkit/com.docker.slirp.tgz \
		| tar xz --strip=2 -C $(dir $@) Contents/MacOS/com.docker.slirp
	touch $@
endif

bin/regextract: | bin
	curl -fsSL https://circleci.com/api/v1/project/justincormack/regextract/latest/artifacts/0/\$$CIRCLE_ARTIFACTS/darwin/amd64/regextract > $@
	chmod a+x $@

hyperkit: scripts/hyperkit.sh bin/com.docker.hyperkit bin/com.docker.slirp alpine/initrd.img kernel/x86_64/vmlinuz64
	./scripts/hyperkit.sh

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
endef

test-hyperkit: scripts/hyperkit.sh bin/com.docker.hyperkit bin/com.docker.slirp alpine/initrd-test.img kernel/x86_64/vmlinuz64
	rm -f disk.img
	INITRD=alpine/initrd-test.img script -q /dev/null ./scripts/hyperkit.sh | tee test.log
	$(call check_test_log, test.log)

test: alpine/initrd-test.img kernel/x86_64/vmlinuz64
	tar cf - $^ | docker run --rm -i $(QEMU_IMAGE) 2>&1 | tee test.log
	$(call check_test_log, test.log)

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
MOBYLINUX_TAG=alpine/mobylinux.tag
ifdef AUFS
AUFS_PREFIX=aufs-
endif
ifdef LTS4.4
AUFS_PREFIX=lts4.4-
endif
MEDIA_IMAGE=mobylinux/media:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG)
INITRD_IMAGE=mobylinux/mobylinux:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG)
KERNEL_IMAGE=mobylinux/kernel:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG)

MEDIA_TOYBOX=mobylinux/toybox-media:0a26fe5f574e444849983f9c4148ef74b3804d55@sha256:5ac38f77b66deb194c9016591b9b096e81fcdc9f7c3e6d01566294a6b4b4ebd2

Dockerfile.media:
	printf "FROM $(MEDIA_TOYBOX)\nADD . /\n" > $@

MEDIA_TARBALL=Dockerfile.media -C alpine initrd.img initrd-test.img mobylinux-efi.iso mobylinux.efi -C ../kernel/x86_64 vmlinuz64 vmlinux kernel-headers.tar kernel-dev.tar

media: Dockerfile.media alpine/initrd.img alpine/initrd-test.img kernel/x86_64/vmlinuz64 alpine/mobylinux-efi.iso
ifeq ($(STATUS),)
	tar cf - $(MEDIA_TARBALL) | docker build -f Dockerfile.media -t $(MEDIA_IMAGE) -
	docker push $(MEDIA_IMAGE)
	[ -f $(MOBYLINUX_TAG) ]
	docker tag $(shell cat $(MOBYLINUX_TAG)) $(INITRD_IMAGE)
	docker push $(INITRD_IMAGE)
	tar cf - Dockerfile.media -C kernel/x86_64 vmlinuz64 | docker build -f Dockerfile.media -t $(KERNEL_IMAGE) -
	docker push $(KERNEL_IMAGE)
else
	$(error "git not clean")
endif

EBPF_TAG=ebpf/ebpf.tag
EBPF_IMAGE=mobylinux/ebpf:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG)
ebpf: alpine/initrd.img kernel/x86_64/vmlinuz64
ifeq ($(STATUS),)
	[ -f $(EBPF_TAG) ]
	docker tag $(shell cat $(EBPF_TAG)) $(EBPF_IMAGE)
	docker push $(EBPF_IMAGE)
else
	$(error "git not clean")
endif

MEDIA_FILES=kernel/x86_64/vmlinuz64 kernel/x86_64/vmlinux alpine/initrd.img alpine/mobylinux-efi.iso alpine/mobylinux.efi
MEDIA_FILES_OPT=kernel/x86_64/kernel-headers.tar kernel/x86_64/kernel-dev.tar alpine/initrd-test.img

get:
ifeq ($(STATUS),)
	IMAGE=$$( docker create mobylinux/media:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG) /dev/null ) && \
	mkdir -p kernel/x86_64 && \
	for FILE in $(MEDIA_FILES); do docker cp $$IMAGE:$$(basename $$FILE) $$FILE || exit; done; \
	for FILE in $(MEDIA_FILES_OPT); do docker cp $$IMAGE:$$(basename $$FILE) $$FILE; done; \
	docker rm $$IMAGE
else
	$(error "git not clean")
endif

# Get artifacts using regextract for cases where docker is not available
get-regextract: bin/regextract
ifeq ($(STATUS),)
	TMP_EXTRACT=$$(mktemp -d) && \
	for FILE in $(MEDIA_FILES) $(MEDIA_FILES_OPT); do mkdir -p $$(dirname $$FILE); done; \
	bin/regextract mobylinux/media:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG) | tar xf - -C $$TMP_EXTRACT && \
	mkdir -p kernel/x86_64 && \
	for FILE in $(MEDIA_FILES); do cp $$TMP_EXTRACT/$$(basename $$FILE) $$FILE || exit; done; \
	for FILE in $(MEDIA_FILES_OPT); do cp $$TMP_EXTRACT/$$(basename $$FILE) $$FILE; done; \
	rm -Rf $$TMP_EXTRACT
else
	$(error "git not clean")
endif

ci:
	$(MAKE) clean
	$(MAKE) all
	$(MAKE) test
	$(MAKE) media

ci-pr:
	$(MAKE) clean
	$(MAKE) all
	$(MAKE) test

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C kernel clean
	rm -rf bin disk.img test.log Dockerfile.media
