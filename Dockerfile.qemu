FROM justincormack/alpine-qemu

COPY alpine/initrd.img .
COPY alpine/kernel/vmlinuz64 .

RUN gzip -9 initrd.img

ENTRYPOINT [ "qemu-system-x86_64", "-serial", "stdio", "-kernel", "vmlinuz64", "-initrd", "initrd.img.gz", "-m", "256", "-append", "earlyprintk=serial console=ttyS0 noapic", "-vnc", "none" ]
