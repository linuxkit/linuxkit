all:	vmlinuz64 aufs-utils.tar

kernel: kernel_config Dockerfile
	docker build -t mobykernel:build .

vmlinuz64: kernel
	docker run mobykernel:build cat /linux/arch/x86_64/boot/bzImage > vmlinuz64

aufs-utils.tar: kernel
	docker run mobykernel:build cat /aufs-utils.tar > aufs-utils.tar

clean:
	rm -f vmlinuz64 aufs-utils.tar
