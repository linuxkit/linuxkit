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

hyperkit.git:
	git clone https://github.com/docker/hyperkit.git hyperkit.git

hyperkit.git/build/com.docker.hyperkit: hyperkit.git
	cd hyperkit.git && make

hyperkit: hyperkit.sh hyperkit.git/build/com.docker.hyperkit alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
	sudo ./hyperkit.sh

test: Dockerfile.test alpine/initrd.img alpine/kernel/x86_64/vmlinuz64
	$(MAKE) -C alpine
	BUILD=$$( tar cf - $^ | docker build -f Dockerfile.test -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	touch test.log && \
	docker run --rm $$BUILD 2>&1 | tee -a test.log
	@cat test.log | grep -q 'Moby test suite PASSED'

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(DOCKER_EXPERIMENTAL),1)
MEDIA_PREFIX?=experimental-
endif
media: Dockerfile.media alpine/initrd.img alpine/kernel/x86_64/vmlinuz64 alpine/mobylinux-efi.iso
ifeq ($(STATUS),)
	tar cf - $^ alpine/mobylinux.efi | docker build -f Dockerfile.media -t mobylinux/media:$(MEDIA_PREFIX)$(TAG) -
	docker push mobylinux/media:$(MEDIA_PREFIX)$(TAG)
else
	$(error "git not clean")
endif

.PHONY: clean

clean:
	$(MAKE) -C alpine clean
	rm -rf hyperkit.git disk.img
