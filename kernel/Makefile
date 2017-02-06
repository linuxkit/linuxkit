DEBUG ?= 0

TAR2INITRD_IMAGE=mobylinux/tar2initrd:d5711601eb5b89de0f052d87365e18388ff3f1b5@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

all:	x86_64/vmlinuz64

x86_64/kernel.img:
	tar cf - etc lib usr sbin | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

ifdef AUFS
x86_64/vmlinuz64: Dockerfile.aufs kernel_config kernel_config.debug kernel_config.aufs patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build -f Dockerfile.aufs --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat aufs-utils.tar | tar xf - && \
	docker run --rm --net=none --log-driver=none $$BUILD cat kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD tar cf - bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar | tar xf - -C x86_64
	mv x86_64/bzImage $@
	rm -rf etc/kernel-patches
	mkdir -p etc/kernel-patches
	cp -a patches-4.9/* etc/kernel-patches
	tar xf x86_64/kernel-modules.tar
	$(MAKE) x86_64/kernel.img
else
x86_64/vmlinuz64: Dockerfile kernel_config kernel_config.debug patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD tar cf - bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar | tar xf - -C x86_64
	mv x86_64/bzImage $@
	rm -rf etc/kernel-patches
	mkdir -p etc/kernel-patches
	cp -a patches-4.9/* etc/kernel-patches
	tar xf x86_64/kernel-modules.tar
	$(MAKE) x86_64/kernel.img
endif

clean:
	rm -rf x86_64 lib etc usr sbin

.DELETE_ON_ERROR:
