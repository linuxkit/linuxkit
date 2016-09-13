DEBUG ?= 0

all:	x86_64/vmlinuz64

x86_64/vmlinuz64: Dockerfile kernel_config
	mkdir -p $(ARCH) etc
	docker build --build-arg DEBUG=$(DEBUG) -t mobykernel:build .
	docker run --rm --net=none mobykernel:build cat /kernel-modules.tar | tar xf -
	docker run --rm --net=none mobykernel:build cat /aufs-utils.tar | tar xf -
	docker run --rm --net=none mobykernel:build cat /kernel-source-info > etc/kernel-source-info
	docker run --rm --net=none mobykernel:build cat /linux/arch/x86_64/boot/bzImage > $@
	cp -a patches etc/kernel-patches

clean:
	rm -rf x86_64 lib etc usr sbin

.DELETE_ON_ERROR:
