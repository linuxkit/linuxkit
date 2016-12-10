all:	initrd.img initrd-test.img mobylinux-efi.iso mobylinux-bios.iso

ETCFILES=$(shell find etc)

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(STATUS),)
DIRTY=
else
DIRTY=-dirty
endif

# Tag: e6cb3f313db7098a1cd21051e678b01931a037a0
ALPINE_BIOS_DIGEST=b06567c9d00fd4d1193e58fa8242a85121482eb2fb20ac4442388b7eb9cdceb3

moby.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	$(MAKE) -C kernel
	$(MAKE) -j -C packages
	printf $(TAG)$(DIRTY) > etc/moby-commit
	BUILD=$$( tar cf - \
	  Dockerfile etc usr init mkinitrd.sh \
	  -C kernel usr etc sbin lib -C .. \
	  -C packages/proxy usr sbin etc -C ../.. \
	  -C packages/transfused sbin etc -C ../.. \
	  -C packages/tap-vsockd sbin etc -C ../.. \
	  -C packages/docker usr etc -C ../.. \
	  -C packages/diagnostics usr etc -C ../.. \
	  -C packages/automount etc -C ../.. \
	  -C packages/hostsettings etc -C ../.. \
	  -C packages/chronyd etc -C ../.. \
	  -C packages/userns etc -C ../.. \
	  -C packages/nc-vsock usr -C ../.. \
	  -C packages/vsudd sbin etc -C ../.. \
	  -C packages/mobyconfig usr -C ../.. \
	  -C packages/mobyplatform usr -C ../.. \
	  -C packages/oom etc -C ../.. \
	  -C packages/9pmount-vsock sbin -C ../.. \
	  -C packages/test etc -C ../.. \
	  -C packages/iptables usr -C ../.. \
	  -C packages/containerd etc -C ../.. \
	  -C packages/aws etc -C ../.. \
	  -C packages/azure etc -C ../.. \
	  | \
	  docker build -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	echo $$BUILD > mobylinux.tag && \
	docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp --tmpfs /initrd $$BUILD > $@

container.img:
	$(MAKE) -j -C containers
	(find containers -type d -maxdepth 1 && \
	 find containers/*/rootfs containers/*/config.json) | \
		cpio -H newc -o | gzip -9 > $@
	SIZE=$$(cat $@ | wc -c); \
	SIZE4=$$(( $$SIZE / 4 * 4 )); \
	DIFF=$$(( $$SIZE - $$SIZE4 )); \
	[ $$DIFF -ne 0 ] && DIFF=$$(( 4 - $$DIFF )); \
	dd if=/dev/zero bs=1 count=$$DIFF of=zeropad
	cat zeropad >> $@
	rm zeropad

test.img:
	$(MAKE) -j -C test
	(find test -maxdepth 0 && \
	 find test/rootfs test/config.json) | \
		cpio -H newc -o | gzip -9 > $@
	SIZE=$$(cat $@ | wc -c); \
	SIZE4=$$(( $$SIZE / 4 * 4 )); \
	DIFF=$$(( $$SIZE - $$SIZE4 )); \
	[ $$DIFF -ne 0 ] && DIFF=$$(( 4 - $$DIFF )); \
	dd if=/dev/zero bs=1 count=$$DIFF of=zeropad
	cat zeropad >> $@
	rm zeropad

initrd.img: moby.img container.img
	cat $^ > $@

initrd-test.img: initrd.img test.img
	cat $^ > $@

mobylinux-efi.iso: Dockerfile.efi initrd.img kernel/x86_64/vmlinuz64
	BUILD=$$( tar cf - $^ | docker build -q -f Dockerfile.efi - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --rm --net=none --log-driver=none --cap-add sys_admin $$BUILD cat /tmp/efi/mobylinux.efi > mobylinux.efi && \
	docker run --rm --net=none --log-driver=none --cap-add sys_admin $$BUILD cat /tmp/efi/mobylinux-efi.iso > $@

mobylinux-bios.iso: initrd.img kernel/x86_64/vmlinuz64
	tar cf - initrd.img -C kernel/x86_64 vmlinuz64 | \
	  docker run --rm --net=none --log-driver=none -i mobylinux/alpine-bios@sha256:$(ALPINE_BIOS_DIGEST) >$@

common: initrd.img
	$(MAKE) -C kernel
	$(MAKE) -j -C packages
	$(MAKE) -j -C containers

ami: common
	tar cf - \
	  cloud initrd.img kernel/x86_64/vmlinuz64 \
	  | \
	  docker build -t moby-ami:build -f cloud/Dockerfile.ami -
	# The EBS device seems not to show up without mounting in /dev, even
	# with --privileged enabled.
	docker run \
		--rm \
		--privileged \
		-v /dev:/dev \
		-e AWS_SECRET_ACCESS_KEY \
		-e AWS_ACCESS_KEY_ID \
		-e TAG_KEY \
		-e TAG_KEY_PREV \
		-e CHANNEL \
		-e MOBY_SRC_ROOT \
		-e DOCKER_BIN_URL \
		moby-ami:build clean
	docker run \
		--rm \
		--privileged \
		-v /dev:/dev \
		-e AWS_SECRET_ACCESS_KEY \
		-e AWS_ACCESS_KEY_ID \
		-e TAG_KEY \
		-e TAG_KEY_PREV \
		-e CHANNEL \
		-e MOBY_SRC_ROOT \
		-e DOCKER_BIN_URL \
		moby-ami:build bake >./cloud/aws/ami_id.out

ami-clean-mount:
	docker run \
		--rm \
		--privileged \
		-v /dev:/dev \
		-e AWS_SECRET_ACCESS_KEY \
		-e AWS_ACCESS_KEY_ID \
		-e TAG_KEY \
		-e TAG_KEY_PREV \
		-e CHANNEL \
		-e MOBY_SRC_ROOT \
		-e DOCKER_BIN_URL \
		moby-ami:build clean-mount

# TODO(nathanleclaire): Migrate this to docker/editions repo.
uploadvhd: azure
	docker run \
		-i \
		-e VHD_SIZE \
		-e AZURE_STG_ACCOUNT_KEY \
		-e AZURE_STG_ACCOUNT_NAME \
		-e CONTAINER_NAME \
		--log-driver none \
		--rm \
		-v vhdartifact:/tmp \
		moby-azure:build \
		uploadvhd >./cloud/azure/vhd_blob_url.out

azure: common vhdartifact
	tar cf - \
	  cloud initrd.img kernel/x86_64/vmlinuz64 \
	  | \
	  docker build -t moby-azure:build -f cloud/Dockerfile.azure -
	tar cf - \
	  cloud \
	  | \
	  docker build -t moby-azure:raw2vhd -f cloud/Dockerfile.raw2vhd -
	# -v /dev:/dev needed in addition to --privileged due to creation of
	#  loopback device (mount namespace?)
	docker run \
		--rm \
		--privileged \
		--log-driver none \
		-v vhdartifact:/tmp \
		-v /dev:/dev \
		moby-azure:build \
		makeraw
	docker run \
		--rm \
		--log-driver none \
		-v vhdartifact:/tmp \
		moby-azure:raw2vhd
	docker run \
		--rm \
		-i \
		--log-driver none \
		-v vhdartifact:/tmp \
		moby-azure:build \
		tarout \
	| tar -xvf -

vhdartifact:
	# NB: Multiple 'docker volume create' with same name does not return
	# non-zero even though maybe it should.  The '|| true' is included as
	# future insurance.
	docker volume create --name vhdartifact || true

clean:
	rm -f *.img *.vhd *.iso *.tag mobylinux.efi zeropad etc/moby-commit
	docker images -q moby-azure:build | xargs docker rmi -f || true
	docker images -q moby-azure:raw2vhd | xargs docker rmi -f || true
	docker volume rm vhdartifact || true
	$(MAKE) -C packages clean
	$(MAKE) -C containers clean
	$(MAKE) -C test clean
	$(MAKE) -C kernel clean

.DELETE_ON_ERROR:
