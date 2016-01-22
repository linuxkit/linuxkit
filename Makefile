all:
	$(MAKE) -C alpine/kernel
	$(MAKE) -C alpine

xhyve: all
	$(MAKE) -C xhyve run

qemu: all
	docker build -f Dockerfile.qemu -t mobyqemu:build .
	docker run -it mobyqemu:build

arm:
	$(MAKE) -C alpine/kernel arm
	$(MAKE) -C alpine arm

qemu-arm: Dockerfile.armhf arm
	docker build -f Dockerfile.qemu.armhf -t mobyarmqemu:build .
	docker run -it mobyarmqemu:build

.PHONY: remora clean
remora: Dockerfile.remora remora/do-kexec.sh
	docker build -f Dockerfile.remora -t justincormack/remora .

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
	docker images -q mobyqemu:build | xargs docker rmi -f
	docker images -q justincormack/remora | xargs docker rmi -f
