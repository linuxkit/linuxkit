ARCH = x86_64
DEBUG ?= 0

all:	$(ARCH)/vmlinuz64

$(ARCH)/mobykernel-build: Dockerfile kernel_config
	mkdir -p $(ARCH) && \
	docker build --build-arg DEBUG=$(DEBUG) -t mobykernel:build .
	touch $@

$(ARCH)/bzImage: $(ARCH)/mobykernel-build
	docker run --rm mobykernel:build cat /linux/arch/$(ARCH)/boot/$(notdir $@) > $@ || ! rm $@
	docker run --rm --net=none mobykernel:build cat /kernel-modules.tar | tar xf -
	docker run --rm mobykernel:build cat /aufs-utils.tar | tar xf -
	docker run --rm mobykernel:build cat /kernel-source-info > etc/kernel-source-info
	mkdir -p etc
	cp -a patches etc/kernel-patches

$(ARCH)/vmlinuz64: $(ARCH)/bzImage
	cp $< $@

clean:
	rm -rf $(ARCH) lib etc usr sbin
	docker images -q mobykernel:build | xargs docker rmi -f || true
