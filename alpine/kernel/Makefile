ARCH ?= x86_64
DEBUG ?= 0

all:	$(ARCH)/vmlinuz64

$(ARCH)/mobykernel-build: Dockerfile kernel_config.$(ARCH)
	mkdir -p $(ARCH) && \
	docker build --build-arg ARCH=$(ARCH) --build-arg DEBUG=$(DEBUG) \
		-t mobykernel-$(ARCH):build .
	touch $@

$(ARCH)/aufs-utils.tar $(ARCH)/kernel-source-info $(ARCH)/kernel-patches.tar $(ARCH)/kernel-modules.tar: $(ARCH)/mobykernel-build
	docker run --rm mobykernel-$(ARCH):build cat /$(notdir $@) > $@ || ! rm $@

$(ARCH)/bzImage $(ARCH)/zImage: $(ARCH)/aufs-utils.tar $(ARCH)/kernel-source-info $(ARCH)/kernel-patches.tar $(ARCH)/kernel-modules.tar $(ARCH)/mobykernel-build
	docker run --rm mobykernel-$(ARCH):build cat /linux/arch/$(ARCH)/boot/$(notdir $@) > $@ || ! rm $@

$(ARCH)/vmlinux: $(ARCH)/bzImage
	docker run --rm mobykernel-$(ARCH):build cat /linux/vmlinux > $@ || ! rm $@

$(ARCH)/vmlinuz64: $(ARCH)/bzImage
	cp $< $@

arm: arm/zImage

clean-arch:
	rm -rf $(ARCH)
	docker images -q mobykernel-$(ARCH):build | xargs docker rmi -f || true

clean:
	$(MAKE) clean-arch ARCH=arm
	$(MAKE) clean-arch ARCH=x86_64
