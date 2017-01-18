DEBUG ?= 0

all:	x86_64/vmlinuz64

ifdef AUFS
x86_64/vmlinuz64: Dockerfile.aufs kernel_config kernel_config.debug kernel_config.aufs patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build -f Dockerfile.aufs --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-modules.tar > x86_64/kernel-modules.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /aufs-utils.tar | tar xf - && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /vmlinux > x86_64/vmlinux && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-headers.tar > x86_64/kernel-headers.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-dev.tar > x86_64/kernel-dev.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /bzImage > $@ && \
	cp -a patches-4.9 etc/kernel-patches && \
	tar xf x86_64/kernel-modules.tar
else
ifdef LTS4.4
x86_64/vmlinuz64: Dockerfile.4.4 kernel_config kernel_config.debug kernel_config.4.4 patches-4.4
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build -f Dockerfile.4.4 --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-modules.tar > x86_64/kernel-modules.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /vmlinux > x86_64/vmlinux && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-headers.tar > x86_64/kernel-headers.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-dev.tar > x86_64/kernel-dev.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /bzImage > $@ && \
	cp -a patches-4.4 etc/kernel-patches && \
	tar xf x86_64/kernel-modules.tar
else
x86_64/vmlinuz64: Dockerfile kernel_config kernel_config.debug patches-4.9
	mkdir -p x86_64 etc lib usr sbin
	BUILD=$$( tar cf - $^ | docker build --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-modules.tar > x86_64/kernel-modules.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-source-info > etc/kernel-source-info && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /vmlinux > x86_64/vmlinux && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-headers.tar > x86_64/kernel-headers.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /kernel-dev.tar > x86_64/kernel-dev.tar && \
	docker run --rm --net=none --log-driver=none $$BUILD cat /bzImage > $@ && \
	cp -a patches-4.9 etc/kernel-patches && \
	tar xf x86_64/kernel-modules.tar
endif
endif

clean:
	rm -rf x86_64 lib etc usr sbin

.DELETE_ON_ERROR:
