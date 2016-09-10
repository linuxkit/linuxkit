all:	initrd.img.gz mobylinux-efi.iso

ETCFILES=etc/issue etc/motd etc/network/interfaces
ETCFILES+=etc/inittab etc/fstab

initrd.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	rm -f initrd.img
	$(MAKE) -j -C packages
	$(MAKE) -C kernel
	tar cf - \
	  Dockerfile etc init mkinitrd.sh \
	  -C kernel usr etc sbin lib -C .. \
	  -C packages/proxy usr sbin etc -C ../.. \
	  -C packages/transfused sbin etc -C ../.. \
	  -C packages/tap-vsockd sbin etc -C ../.. \
	  -C packages/docker usr etc -C ../.. \
	  -C packages/diagnostics usr etc -C ../.. \
	  -C packages/automount etc -C ../.. \
	  -C packages/binfmt_misc etc -C ../.. \
	  -C packages/hostsettings etc -C ../.. \
	  -C packages/hvtools sbin etc usr -C ../.. \
	  -C packages/chronyd etc -C ../.. \
	  -C packages/userns etc -C ../.. \
	  -C packages/nc-vsock usr -C ../.. \
	  -C packages/vsudd sbin etc -C ../.. \
	  -C packages/mobyconfig usr -C ../.. \
	  -C packages/mobyplatform usr -C ../.. \
	  -C packages/oom etc -C ../.. \
	  -C packages/9pmount-vsock sbin -C ../.. \
	  -C packages/test etc usr -C ../.. \
	  -C packages/iptables usr -C ../.. \
	  -C packages/containerd etc -C ../.. \
	  -C packages/aws etc -C ../.. \
	  -C packages/azure etc -C ../.. \
	  | \
	  docker build -t moby-initrd:build -
	docker run --net=none --rm moby-initrd:build /mkinitrd.sh > $@

initrd.img.gz: initrd.img
	cat initrd.img | gzip -9 > initrd.img.gz

mobylinux-efi.iso: initrd.img.gz Dockerfile.efi
	docker build -t moby-efi:build -f Dockerfile.efi .
	docker run --net=none --rm --cap-add sys_admin moby-efi:build cat /tmp/efi/mobylinux.efi > mobylinux.efi
	docker run --net=none --rm --cap-add sys_admin moby-efi:build cat /tmp/efi/mobylinux-efi.iso > $@

mobylinux-bios.iso: initrd.img Dockerfile.bios isolinux.cfg
	docker build -t moby-bios:build -f Dockerfile.bios .
	docker run --net=none --rm moby-bios:build cat /tmp/mobylinux-bios.iso > $@

arm:	initrd-arm.img

Dockerfile.armhf: Dockerfile
	cat Dockerfile | \
	  sed 's@FROM alpine@FROM justincormack/armhf-alpine@' | \
	  sed '/hv_/d' | \
	  sed '/hvtools/d' | \
	  sed 's/syslinux//' | \
	  sed '/proxy/d' | \
	  sed '/diagnostics/d' | \
	  sed '/nc-vsock/d' | \
	  sed '/vsudd/d' | \
	  sed '/gummiboot/d' | \
	  grep -v 'tap-vsockd' | \
	  grep -v '9pmount-vsock' | \
	  grep -v 'rc-update add binfmt_misc sysinit' > $@

initrd-arm.img: Dockerfile.armhf
	rm -f initrd-arm.img
	$(MAKE) -C packages arm
	$(MAKE) -C kernel arm
	docker build -t moby-arm:build -f Dockerfile.armhf .
	docker run --net=none --rm moby-arm:build /bin/mkinitrd.sh > $@

common: initrd.img
	$(MAKE) -C kernel
	$(MAKE) -C packages

ami: common
	docker-compose build ami
	docker-compose run --rm -T ami clean
	docker-compose run --rm -T ami bake

azure: common
	docker-compose build azure
	docker-compose run --rm -T azure makeraw
	docker build -t raw2vhd -f cloud/Dockerfile.raw2vhd cloud
	docker run -v $(shell pwd):/mnt raw2vhd /mnt/mobylinux.img /mnt/mobylinux.vhd
	docker-compose run --rm -T azure uploadvhd

clean:
	rm -f initrd.img initrd.img.gz initrd-arm.img Dockerfile.armhf mobylinux.vhd mobylinux.img
	rm -f mobylinux-bios.iso mobylinux-efi.iso mobylinux.efi
	docker images -q moby-efi:build | xargs docker rmi -f || true
	docker images -q moby-bios:build | xargs docker rmi -f || true
	docker images -q moby-arm:build | xargs docker rmi -f || true
	docker images -q alpine_ami:latest | xargs docker rmi -f || true
	docker images -q alpine_azure:latest | xargs docker rmi -f || true
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean

.DELETE_ON_ERROR:
