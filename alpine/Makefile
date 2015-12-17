all:	initrd.img

ETCFILES=etc/issue etc/motd etc/inittab etc/network/interfaces
ETCFILES+=etc/hostname- etc/resolv.conf- etc/hosts-
ETCFILES+=etc/init.d/chronyd
ETCFILES+=etc/init.d/9pinit etc/init.d/9pudc etc/init.d/mdnstool
ETCFILES+=etc/init.d/automount
ETCFILES+=etc/conf.d/docker
ETCFILES+=etc/securetty

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -C packages/9pudc
	$(MAKE) -C packages/mdnstool
	docker build -t moby:test .
	docker run moby:test /bin/mkinitrd.sh > $@

clean:
	rm -f initrd.img
	$(MAKE) -C packages/9pudc clean
	$(MAKE) -C packages/mdnstool clean
