all:	vmlinuz64

mobykernel-build: Dockerfile kernel_config
	docker build -t mobykernel:build .
	touch $@

mobyarmkernel-build: Dockerfile kernel_config.arm
	docker build --build-arg ARCH=arm -t mobyarmkernel:build .
	touch $@

aufs-utils.tar kernel-source-info kernel-patches.tar kernel-modules.tar: mobykernel-build
	docker run --rm mobykernel:build cat /$@ > $@ || ! rm $@

vmlinuz64: aufs-utils.tar kernel-source-info kernel-patches.tar kernel-modules.tar mobykernel-build
	docker run --rm mobykernel:build cat /linux/arch/x86_64/boot/bzImage > $@ || ! rm $@

arm: zImage

zImage: mobyarmkernel-build aufs-utils.tar kernel-source-info kernel-patches.tar kernel-modules.tar
	docker run --rm mobyarmkernel:build cat /linux/arch/arm/boot/zImage > $@

clean:
	rm -f zImage vmlinuz64 aufs-utils.tar kernel-source-info kernel-patches.tar mobykernel-build mobyarmkernel-build
	docker images -q mobykernel:build | xargs docker rmi -f || true
	docker images -q mobyarmkernel:build | xargs docker rmi -f || true
