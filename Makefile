all:
	$(MAKE) -C alpine/kernel
	$(MAKE) -C alpine

xhyve: all
	$(MAKE) -C xhyve run

qemu: all
	docker build -f Dockerfile.qemu -t mobyqemu:build .
	docker run -it --rm mobyqemu:build

qemu-iso: all
	$(MAKE) -C alpine mobylinux.iso
	docker build -f Dockerfile.qemuiso -t mobyqemuiso:build .
	docker run -it --rm mobyqemuiso:build

arm:
	$(MAKE) -C alpine/kernel arm
	$(MAKE) -C alpine arm

qemu-arm: Dockerfile.qemu.armhf arm
	docker build -f Dockerfile.qemu.armhf -t mobyarmqemu:build .
	docker run -it --rm mobyarmqemu:build

test: Dockerfile.test all
	docker build -f Dockerfile.test -t mobytest:build .
	touch test.log
	docker run --rm mobytest:build 2>&1 | tee -a test.log &
	tail -f test.log 2>/dev/null | grep -m 1 -q 'Moby test suite '
	cat test.log | grep -q 'Moby test suite PASSED'

pull:
	docker pull mobylinux/alpine-build-c
	docker pull mobylinux/alpine-build-go
	docker pull mobylinux/alpine-build-ocaml
	docker pull mobylinux/kernel-build

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
	docker images -q mobyqemu:build | xargs docker rmi -f || true
	docker images -q mobytest:build | xargs docker rmi -f || true
