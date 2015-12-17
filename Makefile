all:
	$(MAKE) -C alpine/kernel
	$(MAKE) -C alpine

xhyve: all
	$(MAKE) -C xhyve run

clean:
	$(MAKE) -C alpine clean
	$(MAKE) -C xhyve clean
