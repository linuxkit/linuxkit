all:	initrd.img

ETCFILES=etc/issue etc/motd etc/inittab etc/network/interfaces
ETCFILES+=etc/hostname- etc/resolv.conf- etc/hosts-
ETCFILES+=etc/securetty

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -C packages
	$(MAKE) -C kernel
	docker build -t mobyinitrd:build .
	docker run mobyinitrd:build /bin/mkinitrd.sh > $@

arm:	initrd-arm.img

Dockerfile.armhf: Dockerfile
	cat Dockerfile | sed 's@FROM alpine@FROM justincormack/armhf-alpine@' | sed 's/syslinux//' > $@

initrd-arm.img: Dockerfile.armhf
	rm -f initrd-arm.img
	$(MAKE) -C packages arm
	$(MAKE) -C kernel arm
	docker build -f Dockerfile.armhf -t mobyinitrdarm:build .
	docker run mobyinitrdarm:build /bin/mkinitrd.sh > $@

clean:
	rm -f initrd.img initrd-arm.img Dockerfile.armhf
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean
	docker images -q mobyinitrd:build | xargs docker rmi -f
