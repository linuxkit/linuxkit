all:	initrd.img initrd-test.img mobylinux-efi.iso mobylinux-bios.iso

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(STATUS),)
DIRTY=
else
DIRTY=-dirty
endif

# By default we want to always auth to GCE to upload moby images
FORCE_GSUTIL_AUTH ?= 1

BIOS_IMAGE=mobylinux/mkimage-iso-bios:2a860edda12a44c8e141a924f29ea931dbc01110@sha256:974304cfff80524a37bf96d299de9b3163ba07df5feeb620c6bdcd132b46f16d

EFI_IMAGE=mobylinux/mkimage-iso-efi:f81034d118744a42e8e93cfe0777dc3490c1f6a0@sha256:4bb10d794fd0fe58366c36ab0fabf7b6d22e5bd760a994900da3b910230519cc

PAD4_IMAGE=mobylinux/pad4:1edffcbfa13d4795f006d38e871a778ffba03d8a@sha256:1ad26970698670373ee0bf374a06900f712a61b8038255e78271b840a1267b25

TAR2INITRD_IMAGE=mobylinux/tar2initrd:d5711601eb5b89de0f052d87365e18388ff3f1b5@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

TARTAR2INITRD_IMAGE=mobylinux/tartar2initrd:d56cde1558e3080e59a32e3cd7c7141baa601811@sha256:e1ad4522ff906d339da5f250b9ef6bffa5a70b4dec7d2cf7f7dbd0447b79352f

MKIMAGE_BASE=mobylinux/mkimage-base:870f7512498f2ce5feccebe15fb0d03c5c3ebac2@sha256:47d1ed872b6a44f13b61ea80b3eeab4519dc151c7d684a89a53aa26233b4e087

moby.img: Dockerfile etc usr init
	$(MAKE) -j -C packages
	printf $(TAG)$(DIRTY) > etc/moby-commit
	BUILD=$$( tar cf - $^ \
	  -C packages/proxy usr sbin etc -C ../.. \
	  -C packages/transfused sbin etc -C ../.. \
	  -C packages/tap-vsockd sbin etc -C ../.. \
	  -C packages/docker usr etc -C ../.. \
	  -C packages/diagnostics usr etc -C ../.. \
	  -C packages/automount etc -C ../.. \
	  -C packages/windowsnet etc -C ../.. \
	  -C packages/hostsettings etc -C ../.. \
	  -C packages/chronyd etc -C ../.. \
	  -C packages/userns etc -C ../.. \
	  -C packages/nc-vsock usr -C ../.. \
	  -C packages/vsudd sbin etc -C ../.. \
	  -C packages/mobyconfig etc usr -C ../.. \
	  -C packages/mobyplatform usr -C ../.. \
	  -C packages/oom etc -C ../.. \
	  -C packages/9pmount-vsock sbin -C ../.. \
	  -C packages/test etc -C ../.. \
	  -C packages/iptables usr -C ../.. \
	  -C packages/containerd etc -C ../.. \
	  -C packages/aws etc -C ../.. \
	  -C packages/azure etc -C ../.. \
	  -C packages/gcp etc -C ../.. \
	  | \
	  docker build -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	echo $$BUILD > mobylinux.tag && \
	docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -v /var/run/docker.sock:/var/run/docker.sock $(MKIMAGE_BASE) $$BUILD | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

container.img:
	$(MAKE) -C containers
	tar cf - containers/*/container.tar | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TARTAR2INITRD_IMAGE) > $@

test/test.img:
	$(MAKE) -C test

../kernel/x86_64/kernel.img:
	$(MAKE) -C ../kernel

initrd.img: moby.img ../kernel/x86_64/kernel.img container.img
	cat $^ > $@

initrd-test.img: initrd.img test/test.img
	cat $^ > $@

# outputs tarball of mobylinux-efi.iso mobylinux.efi
mobylinux-efi.iso: initrd.img ../kernel/x86_64/vmlinuz64
	tar cf - initrd.img -C ../kernel/x86_64 vmlinuz64 | docker run --rm --net=none --log-driver=none -i $(EFI_IMAGE) | tar xf -

mobylinux-bios.iso: initrd.img ../kernel/x86_64/vmlinuz64
	tar cf - initrd.img -C ../kernel/x86_64 vmlinuz64 | docker run --rm --net=none --log-driver=none -i $(BIOS_IMAGE) >$@

clean:
	rm -f *.img *.vhd *.iso *.tag mobylinux.efi etc/moby-commit
	$(MAKE) -C packages clean
	$(MAKE) -C containers clean
	$(MAKE) -C test clean

.DELETE_ON_ERROR:
