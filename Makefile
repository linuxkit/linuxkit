all:
	$(MAKE) -C alpine

alpine/initrd.img.gz:
	$(MAKE) -C alpine initrd.img.gz

alpine/kernel/x86_64/vmlinuz64:
	$(MAKE) -C alpine/kernel x86_64/vmlinuz64

alpine/mobylinux-bios.iso:
	$(MAKE) -C alpine mobylinux-bios.iso

QEMU_DEPS=Dockerfile.qemu alpine/initrd.img.gz alpine/kernel/x86_64/vmlinuz64
qemu: $(QEMU_DEPS)
	tar cf - $(QEMU_DEPS) | docker build -f Dockerfile.qemu -t mobyqemu:build -
	docker run -it --rm mobyqemu:build

QEMU_ISO_DEPS=Dockerfile.qemuiso alpine/mobylinux-bios.iso
qemu-iso: $(TEST_DEPS)
	tar cf - $(QEMU_ISO_DEPS) | docker build -f Dockerfile.qemuiso -t mobyqemuiso:build -
	docker run -it --rm mobyqemuiso:build

TEST_DEPS=Dockerfile.test alpine/initrd.img.gz alpine/kernel/x86_64/vmlinuz64
test: $(TEST_DEPS)
	tar cf - $(TEST_DEPS) | docker build -f Dockerfile.test -t mobytest:build -
	touch test.log
	docker run --rm mobytest:build 2>&1 | tee -a test.log &
	tail -f test.log 2>/dev/null | grep -m 1 -q 'Moby test suite '
	cat test.log | grep -q 'Moby test suite PASSED'

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	docker images -q mobyqemu:build | xargs docker rmi -f || true
	docker images -q mobytest:build | xargs docker rmi -f || true
