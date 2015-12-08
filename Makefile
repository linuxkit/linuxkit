all:
	$(MAKE) -C kernel
	$(MAKE) -C alpine

xhyve: all
	$(MAKE) -C xhyve run

clean:
	$(MAKE) -C kernel clean
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
