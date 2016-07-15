all:	initrd.img.gz mobylinux-efi.iso

ETCFILES=etc/issue etc/motd etc/network/interfaces
ETCFILES+=etc/inittab

initrd.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -C packages
	$(MAKE) -C kernel
	docker-compose build moby
	docker-compose run --rm -T moby /bin/mkinitrd.sh

initrd.img.gz: initrd.img
	cat initrd.img | gzip -9 > initrd.img.gz

mobylinux-efi.iso: initrd.img.gz Dockerfile.efi
	docker-compose build efi
	docker-compose run --rm -T efi \
		cp /tmp/efi/mobylinux.efi /tmp/efi/mobylinux-efi.iso /mnt

mobylinux-bios.iso: initrd.img Dockerfile.bios isolinux.cfg
	docker-compose build bios
	docker-compose run --rm -T bios \
		cp /tmp/mobylinux-bios.iso /mnt

arm:	initrd-arm.img

Dockerfile.armhf: Dockerfile
	cat Dockerfile | \
	  sed 's@FROM alpine@FROM justincormack/armhf-alpine@' | \
	  sed '/hv_/d' | \
	  sed '/hvtools/d' | \
	  sed 's/syslinux//' | \
	  sed '/proxy/d' | \
	  sed '/llmnrd/d' | \
	  sed '/diagnostics/d' | \
	  sed '/nc-vsock/d' | \
	  sed '/vsudd/d' | \
	  sed '/gummiboot/d' | \
	  grep -v 'rc-update add binfmt_misc sysinit' > $@

initrd-arm.img: Dockerfile.armhf
	rm -f initrd-arm.img
	$(MAKE) -C packages arm
	$(MAKE) -C kernel arm
	cp inittab.armhf etc/inittab
	docker-compose build arm
	docker-compose run --rm -T arm /bin/mkinitrd.sh > $@

ami: initrd.img
	$(MAKE) -C kernel
	$(MAKE) -C packages
	docker-compose build ami
	docker-compose run --rm -T ami clean
	docker-compose run --rm -T ami bake

clean:
	rm -f initrd.img initrd.img.gz initrd-arm.img Dockerfile.armhf etc/inittab
	rm -f mobylinux-bios.iso mobylinux-efi.iso mobylinux.efi
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean
