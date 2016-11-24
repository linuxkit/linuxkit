all:	initrd.img mobylinux-efi.iso mobylinux-bios.iso

ETCFILES=$(shell find etc)

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(STATUS),)
DIRTY=
else
DIRTY=-dirty
endif

initrd.img: Dockerfile mkinitrd.sh init $(ETCFILES)
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
	  -C packages/binfmt_misc etc -C ../.. \
	  -C packages/hostsettings etc -C ../.. \
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
	  docker build -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --net=none --log-driver=none --rm $$BUILD > $@

mobylinux-efi.iso: Dockerfile.efi initrd.img kernel/x86_64/vmlinuz64
	BUILD=$$( tar cf - $^ | docker build -q -f Dockerfile.efi - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	docker run --net=none --log-driver=none --rm --cap-add sys_admin $$BUILD cat /tmp/efi/mobylinux.efi > mobylinux.efi && \
	docker run --net=none --log-driver=none --rm --cap-add sys_admin $$BUILD cat /tmp/efi/mobylinux-efi.iso > $@

mobylinux-bios.iso: initrd.img kernel/x86_64/vmlinuz64
	tar cf - initrd.img -C kernel/x86_64 vmlinuz64 | \
	  docker run --net=none --log-driver=none --rm -i mobylinux/alpine-bios:e6cb3f313db7098a1cd21051e678b01931a037a0 >$@

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
	rm -f initrd.img mobylinux.vhd mobylinux.img mobylinux-bios.iso mobylinux-efi.iso mobylinux.efi etc/moby-commit
	docker images -q moby-azure:build | xargs docker rmi -f || true
	docker images -q moby-azure:raw2vhd | xargs docker rmi -f || true
	docker volume rm vhdartifact || true
	$(MAKE) -C packages clean
	$(MAKE) -C kernel clean

.DELETE_ON_ERROR:
