FROM debian:unstable

COPY alpine/initrd.img .
COPY alpine/kernel/vmlinuz64 .

RUN apt-get update && apt-get -y install qemu

RUN gzip -9 initrd.img

ENTRYPOINT [ "qemu-system-x86_64", "-kernel", "vmlinuz64", "-initrd", "initrd.img.gz", "-append", "earlyprintk=serial console=ttyS0", "-vnc", "none", "-nographic" ]
