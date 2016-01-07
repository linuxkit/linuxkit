all:	initrd.img

ETCFILES=etc/issue etc/motd etc/inittab etc/network/interfaces
ETCFILES+=etc/hostname- etc/resolv.conf- etc/hosts-
ETCFILES+=etc/securetty

kernel/aufs-utils.tar:
	$(MAKE) -C kernel aufs-utils.tar

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES) kernel/aufs-utils.tar
	rm -f initrd.img
	$(MAKE) -C packages
	docker build -t mobyinitrd:build .
	docker run mobyinitrd:build /bin/mkinitrd.sh > $@

clean:
	rm -f initrd.img
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean
	docker images -q mobyinitrd:build | xargs docker rmi -f
