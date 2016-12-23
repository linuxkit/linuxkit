DEBUG ?= 0

all:	x86_64/vmlinuz64

ifdef AUFS
x86_64/vmlinuz64: Dockerfile.aufs kernel_config kernel_config.debug kernel_config.aufs patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build -f Dockerfile.aufs --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-modules.tar | tar xf - && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /linux/vmlinux > x86_64/vmlinux && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /linux/arch/x86_64/boot/bzImage > $@ && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-headers.tar > x86_64/kernel-headers.tar && \
	cp -a patches-4.9 etc/kernel-patches
else
x86_64/vmlinuz64: Dockerfile kernel_config kernel_config.debug patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-modules.tar | tar xf - && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /linux/vmlinux > x86_64/vmlinux && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /linux/arch/x86_64/boot/bzImage > $@ && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-headers.tar > x86_64/kernel-headers.tar && \
	cp -a patches-4.9 etc/kernel-patches
endif

clean:
	rm -rf x86_64 lib etc usr sbin

.DELETE_ON_ERROR:
