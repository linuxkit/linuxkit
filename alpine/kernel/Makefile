all:	vmlinuz64 aufs-utils.tar

vmlinuz64: kernel_config Dockerfile
	docker build -t mobykernel:build .
	docker run mobykernel:build cat /linux/arch/x86_64/boot/bzImage > $@

aufs-utils.tar: vmlinuz64
	docker run mobykernel:build cat /aufs-utils.tar > $@

clean:
	rm -f vmlinuz64 aufs-utils.tar
