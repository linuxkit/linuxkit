all:	vmlinuz64

vmlinuz64:
	docker build -t mobykernel:test .
	docker run mobykernel:test cat /linux/arch/x86_64/boot/bzImage > vmlinuz64

clean:
	rm -f vmlinuz64
