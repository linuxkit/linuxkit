all: qemu-arm-static 

qemu-arm-static :
	docker build -t qemu-user-static:build .
	docker run qemu-user-static:build cat /usr/bin/qemu-arm-static > qemu-arm-static
	chmod 755 qemu-arm-static
	docker run qemu-user-static:build cat /var/lib/binfmts/qemu-arm > binfmt-arm

clean:
	rm -f qemu-arm-static binfmt-arm
