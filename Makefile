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
	docker run --rm mobytest:build | tee test.log | grep 'Moby test suite PASSED'

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
	docker images -q mobyqemu:build | xargs docker rmi -f || true
	docker images -q mobytest:build | xargs docker rmi -f || true
