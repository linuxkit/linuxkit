all:	initrd.img mobylinux-efi.iso mobylinux-bios.iso

ETCFILES=$(shell find etc)

initrd.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	$(MAKE) -C kernel
	$(MAKE) -j -C packages
	tar cf - \
	  Dockerfile etc init mkinitrd.sh \
	  -C kernel usr etc sbin lib -C .. \
	  -C packages/proxy usr sbin etc -C ../.. \
	  -C packages/transfused sbin etc -C ../.. \
	  -C packages/tap-vsockd sbin etc -C ../.. \
	  -C packages/docker usr etc -C ../.. \
	  -C packages/diagnostics usr etc -C ../.. \
	  -C packages/automount etc -C ../.. \
	  -C packages/binfmt_misc etc -C ../.. \
	  -C packages/hostsettings etc -C ../.. \
	  -C packages/hvtools sbin etc usr -C ../.. \
	  -C packages/chronyd etc -C ../.. \
	  -C packages/userns etc -C ../.. \
	  -C packages/nc-vsock usr -C ../.. \
	  -C packages/vsudd sbin etc -C ../.. \
	  -C packages/mobyconfig usr -C ../.. \
	  -C packages/mobyplatform usr -C ../.. \
	  -C packages/oom etc -C ../.. \
	  -C packages/9pmount-vsock sbin -C ../.. \
	  -C packages/test etc usr -C ../.. \
	  -C packages/iptables usr -C ../.. \
	  -C packages/containerd etc -C ../.. \
	  -C packages/aws etc -C ../.. \
	  -C packages/azure etc -C ../.. \
	  | \
	  docker build -t moby-initrd:build -
	docker run --net=none --log-driver=none --rm moby-initrd:build > $@

mobylinux-efi.iso: Dockerfile.efi initrd.img kernel/x86_64/vmlinuz64
	tar cf - $^ | docker build -t moby-efi:build -f Dockerfile.efi -
	docker run --net=none --log-driver=none --rm --cap-add sys_admin moby-efi:build cat /tmp/efi/mobylinux.efi > mobylinux.efi
	docker run --net=none --log-driver=none --rm --cap-add sys_admin moby-efi:build cat /tmp/efi/mobylinux-efi.iso > $@

mobylinux-bios.iso: Dockerfile.bios initrd.img kernel/x86_64/vmlinuz64 isolinux.cfg
	tar cf - $^ | docker build -t moby-bios:build -f Dockerfile.bios -
	docker run --net=none --log-driver=none --rm moby-bios:build cat /tmp/mobylinux-bios.iso > $@

common: initrd.img
	$(MAKE) -C kernel
	$(MAKE) -C packages

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
		-e AZURE_STG_ACCOUNT_KEY \
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
	rm -f initrd.img mobylinux.vhd mobylinux.img mobylinux-bios.iso mobylinux-efi.iso mobylinux.efi
	docker images -q moby-azure:build | xargs docker rmi -f || true
	docker images -q moby-azure:raw2vhd | xargs docker rmi -f || true
	docker volume rm vhdartifact || true
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean

.DELETE_ON_ERROR:
