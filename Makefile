all:
	$(MAKE) -C alpine/kernel
	$(MAKE) -C alpine

xhyve: all
	$(MAKE) -C xhyve run

qemu: all
	docker build -t mobyqemu:build .
	docker run -it mobyqemu:build

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
	docker images -q mobyqemu:build | xargs docker rmi
