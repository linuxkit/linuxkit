all:	initrd.img

ETCFILES=etc/issue etc/motd etc/inittab etc/network/interfaces
#ETCFILES+=etc/init.d/sysfs etc/init.d/cgroupfs-mount
ETCFILES+=etc/hostname- etc/resolv.conf- etc/hosts-

initrd.img: Dockerfile mkinitrd.sh repositories $(ETCFILES)
	rm -f initrd.img
	docker build -t moby:test .
	docker run -i -v $(PWD):/export moby:test /export/mkinitrd.sh

clean:
	rm -f initrd.img
