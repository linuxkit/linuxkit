all:	initrd.img

initrd.img: Dockerfile hostname repositories motd
	rm -f initrd.img
	docker build -t moby:test .
	docker run -i -v $(PWD):/export moby:test /export/mkinitrd.sh

clean:
	rm -f initrd.img
