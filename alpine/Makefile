all:	initrd.img

initrd.img: Dockerfile repositories etc/hostname etc/motd etc/inittab
	rm -f initrd.img
	docker build -t moby:test .
	docker run -i -v $(PWD):/export moby:test /export/mkinitrd.sh

clean:
	rm -f initrd.img
