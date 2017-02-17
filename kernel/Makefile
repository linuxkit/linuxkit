DEBUG ?= 0

TAR2INITRD_IMAGE=mobylinux/tar2initrd:d5711601eb5b89de0f052d87365e18388ff3f1b5@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

all:	x86_64/vmlinuz64

x86_64/kernel.img:
	tar cf - etc lib usr sbin | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

ifdef AUFS
kernel.tag: Dockerfile.aufs kernel_config kernel_config.debug kernel_config.aufs patches-4.9
else
ifdef LTS4.4
kernel.tag: Dockerfile.4.4 kernel_config kernel_config.debug kernel_config.4.4 patches-4.4
else
kernel.tag: Dockerfile kernel_config kernel_config.debug patches-4.9
endif
endif
	BUILD=$$( tar cf - $^ | docker build -f $< --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

x86_64/vmlinuz64: kernel.tag
	rm -rf etc/kernel-patches
	mkdir -p x86_64 etc lib usr sbin etc/kernel-patches
	docker run --rm --net=none --log-driver=none $(shell cat kernel.tag) cat kernel-source-info > etc/kernel-source-info
	docker run --rm --net=none --log-driver=none $(shell cat kernel.tag) tar cf - bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar | tar xf - -C x86_64
	docker run --rm --net=none --log-driver=none $(shell cat kernel.tag) tar cf - -C patches . | tar xf - -C etc/kernel-patches
	mv x86_64/bzImage $@
	tar xf x86_64/kernel-modules.tar
	$(MAKE) x86_64/kernel.img

clean:
	rm -rf x86_64 lib etc usr sbin kernel.tag

.DELETE_ON_ERROR:
