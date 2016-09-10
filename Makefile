all:
	$(MAKE) -C alpine/kernel
	$(MAKE) -C alpine

qemu: all
	tar cf - Dockerfile.qemu alpine/initrd.img.gz alpine/kernel/x86_64/vmlinuz64 | \
	  docker build -f Dockerfile.qemu -t mobyqemu:build -
	docker run -it --rm mobyqemu:build

qemu-iso: all
	$(MAKE) -C alpine mobylinux-bios.iso
	docker build -f Dockerfile.qemuiso -t mobyqemuiso:build .
	docker run -it --rm mobyqemuiso:build

test: Dockerfile.test all
	tar cf - Dockerfile.test alpine/initrd.img.gz alpine/kernel/x86_64/vmlinuz64 | \
	  docker build -f Dockerfile.test -t mobytest:build -
	touch test.log
	docker run --rm mobytest:build 2>&1 | tee -a test.log &
	tail -f test.log 2>/dev/null | grep -m 1 -q 'Moby test suite '
	cat test.log | grep -q 'Moby test suite PASSED'

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	docker images -q mobyqemu:build | xargs docker rmi -f || true
	docker images -q mobytest:build | xargs docker rmi -f || true
