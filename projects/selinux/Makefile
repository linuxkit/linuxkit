MOBY=../../bin/moby

selinux-initrd.img: $(MOBY) selinux.yml
	$^

.PHONY: qemu
qemu: selinux-initrd.img
	../../scripts/qemu.sh selinux-initrd.img selinux-bzImage "$(shell cat selinux-cmdline)"

.PHONY: clean
clean:
	-rm -rf *-initrd.img *-bzImage *-cmdline *.iso *.tar.gz *.qcow2 *.vhd
