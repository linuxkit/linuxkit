all:	initrd.img

ETCFILES=etc/issue etc/motd etc/inittab etc/network/interfaces
ETCFILES+=etc/hostname- etc/resolv.conf- etc/hosts-
ETCFILES+=etc/init.d/chronyd
ETCFILES+=etc/init.d/9pinit etc/init.d/9pudc etc/init.d/mdnstool
ETCFILES+=etc/init.d/automount
ETCFILES+=etc/conf.d/docker
ETCFILES+=etc/securetty

kernel/aufs-utils.tar:
	$(MAKE) -C kernel aufs-utils.tar

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES) kernel/aufs-utils.tar
	rm -f initrd.img
	$(MAKE) -C packages
	docker build -t moby:build .
	docker run moby:build /bin/mkinitrd.sh > $@

clean:
	rm -f initrd.img
	$(MAKE) -C packages clean
