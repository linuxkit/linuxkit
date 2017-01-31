all:
	$(MAKE) -C alpine

aufs:
	$(MAKE) AUFS=true all

alpine/initrd.img:
	$(MAKE) -C alpine initrd.img

alpine/initrd-test.img:
	$(MAKE) -C alpine

alpine/kernel/x86_64/vmlinuz64:
	$(MAKE) -C alpine/kernel x86_64/vmlinuz64

alpine/mobylinux-bios.iso:
	$(MAKE) -C alpine mobylinux-bios.iso

alpine/gce.img.tar.gz:
	$(MAKE) -C alpine gce.img.tar.gz

QEMU_IMAGE=mobylinux/qemu:0fb8c648e8ed9ef6b1ec449587aeab6c53872744@sha256:606f30d815102e73bc01c07915dc0d5f153b0252c63f5f0ed1e39621ec656eb5

# interactive versions need to use volume mounts
qemu: alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
	docker run -it --rm -v $(CURDIR)/alpine/initrd.img:/tmp/initrd.img -v $(CURDIR)/alpine/kernel/x86_64/vmlinuz64:/tmp/vmlinuz64 $(QEMU_IMAGE)

qemu-iso: alpine/mobylinux-bios.iso
	docker run -it --rm -v $(CURDIR)/alpine/mobylinux-bios.iso:/tmp/mobylinux-bios.iso $(QEMU_IMAGE)

qemu-gce: alpine/gce.img.tar.gz
	docker run -it --rm -v $(CURDIR)/alpine/gce.img.tar.gz:/tmp/gce.img.tar.gz $(QEMU_IMAGE)

hyperkit.bin:
	mkdir $@

hyperkit.bin/com.docker.hyperkit: hyperkit.bin
	curl -fsSL https://circleci.com/api/v1/project/docker/hyperkit/latest/artifacts/0//Users/distiller/hyperkit/build/com.docker.hyperkit > $@
	chmod a+x $@

hyperkit.bin/com.docker.slirp:
	curl -fsSL https://circleci.com/api/v1/project/docker/vpnkit/latest/artifacts/0//Users/distiller/vpnkit/com.docker.slirp.tgz \
		| tar xz --strip=2 -C hyperkit.bin Contents/MacOS/com.docker.slirp

hyperkit: hyperkit.sh hyperkit.bin/com.docker.hyperkit hyperkit.bin/com.docker.slirp alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
	./hyperkit.sh

define check_test_log
	@cat $1 |grep -q 'Moby test suite PASSED'
endef

hyperkit-test: hyperkit.sh hyperkit.bin/com.docker.hyperkit hyperkit.bin/com.docker.slirp alpine/initrd-test.img alpine/kernel/x86_64/vmlinuz64
	rm -f disk.img
	INITRD=alpine/initrd-test.img ./hyperkit.sh 2>&1 | tee test.log
	$(call check_test_log, test.log)

test: alpine/initrd-test.img alpine/kernel/x86_64/vmlinuz64
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
media: Dockerfile.media alpine/initrd.img alpine/kernel/x86_64/vmlinuz64 alpine/mobylinux-efi.iso
ifeq ($(STATUS),)
	tar cf - $^ alpine/mobylinux.efi alpine/kernel/x86_64/vmlinux alpine/kernel/x86_64/kernel-headers.tar alpine/kernel/x86_64/kernel-dev.tar | docker build -f Dockerfile.media -t $(MEDIA_IMAGE) -
	docker push $(MEDIA_IMAGE)
	[ -f $(MOBYLINUX_TAG) ]
	docker tag $(shell cat $(MOBYLINUX_TAG)) $(INITRD_IMAGE)
	docker push $(INITRD_IMAGE)
	tar cf - Dockerfile.kernel alpine/kernel/x86_64/vmlinuz64 | docker build -f Dockerfile.kernel -t $(KERNEL_IMAGE) -
	docker push $(KERNEL_IMAGE)
else
	$(error "git not clean")
endif

EBPF_TAG=alpine/base/ebpf/ebpf.tag
EBPF_IMAGE=mobylinux/ebpf:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG)
ebpf: alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
ifeq ($(STATUS),)
	[ -f $(EBPF_TAG) ]
	docker tag $(shell cat $(EBPF_TAG)) $(EBPF_IMAGE)
	docker push $(EBPF_IMAGE)
else
	$(error "git not clean")
endif

get:
ifeq ($(STATUS),)
	IMAGE=$$( docker create mobylinux/media:$(MEDIA_PREFIX)$(AUFS_PREFIX)$(TAG) /dev/null ) && \
	mkdir -p alpine/kernel/x86_64 && \
	docker cp $$IMAGE:vmlinuz64 alpine/kernel/x86_64/vmlinuz64 && \
	docker cp $$IMAGE:vmlinux alpine/kernel/x86_64/vmlinux && \
	(docker cp $$IMAGE:kernel-headers.tar alpine/kernel/x86_64/kernel-headers.tar || true) && \
	(docker cp $$IMAGE:kernel-dev.tar alpine/kernel/x86_64/kernel-dev.tar || true) && \
	docker cp $$IMAGE:initrd.img alpine/initrd.img && \
	docker cp $$IMAGE:mobylinux-efi.iso alpine/mobylinux-efi.iso && \
	docker cp $$IMAGE:mobylinux.efi alpine/mobylinux.efi && \
	docker rm $$IMAGE
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
	rm -rf hyperkit.bin disk.img test.log
