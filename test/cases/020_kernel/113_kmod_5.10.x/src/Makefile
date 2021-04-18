obj-m += hello_world.o
KVER=$(shell basename /usr/src/linux-headers-*)
all:
	make -C /usr/src/$(KVER) M=$(PWD) modules
clean:
	make -C /usr/src/$(KVER) M=$(PWD) clean
