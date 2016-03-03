all:	initrd.img

ETCFILES=etc/issue etc/motd etc/network/interfaces
ETCFILES+=etc/securetty

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -C packages
	$(MAKE) -C kernel
	cp inittab.x86_64 etc/inittab
	docker-compose build moby
	docker-compose run --rm -T moby /bin/mkinitrd.sh

mobylinux.iso: initrd.img Dockerfile.iso isolinux.cfg
	docker-compose run --rm -T iso cp /tmp/output.iso /mnt/mobylinux.iso

arm:	initrd-arm.img

Dockerfile.armhf: Dockerfile
	cat Dockerfile | \
	  sed 's@FROM alpine@FROM justincormack/armhf-alpine@' | \
	  sed '/hv_/d' | \
	  sed '/hvtools/d' | \
	  sed 's/syslinux//' | \
	  grep -v 'rc-update add binfmt_misc sysinit' > $@

initrd-arm.img: Dockerfile.armhf
	rm -f initrd-arm.img
	$(MAKE) -C packages arm
	$(MAKE) -C kernel arm
	cp inittab.armhf etc/inittab
	docker-compose build arm
	docker-compose run --rm -T arm /bin/mkinitrd.sh > $@

clean:
	rm -f initrd.img initrd-arm.img Dockerfile.armhf etc/inittab
	rm -f mobylinux.iso
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean
