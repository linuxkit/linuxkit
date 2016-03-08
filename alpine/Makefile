all:	initrd.img.gz

ETCFILES=etc/issue etc/motd etc/network/interfaces
ETCFILES+=etc/securetty

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -C packages
	$(MAKE) -C kernel
	cp inittab.x86_64 etc/inittab
	docker-compose build moby
	docker-compose run --rm -T moby /bin/mkinitrd.sh

initrd.img.gz: initrd.img
	cat initrd.img | gzip -9 > initrd.img.gz

mobylinux.iso: initrd.img Dockerfile.iso isolinux.cfg
	docker-compose build iso
	docker-compose run --rm -T iso cp /tmp/output.iso /mnt/mobylinux.iso

mobylinuxefi.iso mobylinux.efi:
mobylinux.efi: initrd.img Dockerfile.efi
	docker-compose build efi
	docker-compose run --rm -T efi cp /tmp/efi/mobylinux.efi /tmp/efi/mobylinuxefi.iso /mnt

arm:	initrd-arm.img

Dockerfile.armhf: Dockerfile
	cat Dockerfile | \
	  sed 's!FROM alpine.*!FROM justincormack/armhf-alpine@sha256:e05144d780b6b88c3ae8147d61bb0c7d2b8e62710dae0c41799d400b3620ab36!' | \
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
	rm -f initrd.img initrd.img.gz initrd-arm.img Dockerfile.armhf etc/inittab
	rm -f mobylinux.iso mobylinuxefi.iso mobylinux-boot.vhdx mobylinux.efi
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean
