all:	vmlinuz64

vmlinuz64: kernel_config Dockerfile
	docker build -t mobykernel:build .
	docker run --rm mobykernel:build cat /linux/arch/x86_64/boot/bzImage > $@
	docker run --rm mobykernel:build cat /aufs-utils.tar > aufs-utils.tar
	docker run --rm mobykernel:build cat /kernel-source-info > kernel-source-info
	docker run --rm mobykernel:build cat /kernel-patches.tar > kernel-patches.tar
	docker run --rm mobykernel:build cat /kernel-modules.tar > kernel-modules.tar

arm: zImage

zImage: kernel_config.arm Dockerfile
	docker build --build-arg ARCH=arm -t mobyarmkernel:build .
	docker run --rm mobyarmkernel:build cat /linux/arch/arm/boot/zImage > $@
	docker run --rm mobyarmkernel:build cat /aufs-utils.tar > aufs-utils.tar
	docker run --rm mobyarmkernel:build cat /kernel-source-info > kernel-source-info
	docker run --rm mobyarmkernel:build cat /kernel-patches.tar > kernel-patches.tar
	docker run --rm mobyarmkernel:build cat /kernel-modules.tar > kernel-modules.tar

clean:
	rm -f zImage vmlinuz64 aufs-utils.tar kernel-source-info kernel-patches.tar
	docker images -q mobykernel:build | xargs docker rmi -f || true
	docker images -q mobyarmkernel:build | xargs docker rmi -f || true
