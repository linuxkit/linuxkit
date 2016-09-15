DOCKER_EXPERIMENTAL?=1

all:
	$(MAKE) -C alpine

alpine/initrd.img:
	$(MAKE) -C alpine initrd.img

alpine/kernel/x86_64/vmlinuz64:
	$(MAKE) -C alpine/kernel x86_64/vmlinuz64

alpine/mobylinux-bios.iso:
	$(MAKE) -C alpine mobylinux-bios.iso

qemu: Dockerfile.qemu alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
	tar cf - $^ | docker build -f Dockerfile.qemu -t mobyqemu:build -
	docker run -it --rm mobyqemu:build

qemu-iso: Dockerfile.qemuiso alpine/mobylinux-bios.iso
	tar cf - $^ | docker build -f Dockerfile.qemuiso -t mobyqemuiso:build -
	docker run -it --rm mobyqemuiso:build

test: Dockerfile.test alpine/initrd.img alpine/kernel/x86_64/vmlinuz64 lint
	$(MAKE) -C alpine
	tar cf - $^ | docker build -f Dockerfile.test -t mobytest:build -
	touch test.log
	docker run --rm mobytest:build 2>&1 | tee -a test.log &
	tail -f test.log 2>/dev/null | grep -m 1 -q 'Moby test suite '

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(DOCKER_EXPERIMENTAL),1)
MEDIA_PREFIX?=experimental-
endif
media: Dockerfile.media alpine/initrd.img alpine/kernel/x86_64/vmlinuz64 alpine/mobylinux-bios.iso alpine/mobylinux-efi.iso
ifeq ($(STATUS),)
	tar cf - $^ alpine/mobylinux.efi | docker build -f Dockerfile.media -t mobylinux/media:$(MEDIA_PREFIX)latest -
	docker tag mobylinux/media:$(MEDIA_PREFIX)latest mobylinux/media:$(MEDIA_PREFIX)$(TAG)
	docker push mobylinux/media:$(MEDIA_PREFIX)$(TAG)
	docker push mobylinux/media:$(MEDIA_PREFIX)latest
else
	$(error "git not clean")
endif

.PHONY: clean

clean:
	$(MAKE) -C alpine clean

SCRIPTS=$(shell find . -type f \
	! -path "./*.git/*" \
	! -path "./xhyve/*" \
	! -path "./alpine/cloud/*" \
	-exec file {} \; | grep 'POSIX\|openrc' | cut -d ":" -f 1)

lint:
	@docker run -it --rm -v $(shell pwd):/mnt nlknguyen/alpine-shellcheck:v0.4.4 -e SC1008 ${SCRIPTS}
