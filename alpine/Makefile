all:	initrd.img initrd-test.img mobylinux-efi.iso mobylinux-bios.iso

ETCFILES=$(shell find etc)

TAG=$(shell git rev-parse HEAD)
STATUS=$(shell git status -s)
ifeq ($(STATUS),)
DIRTY=
else
DIRTY=-dirty
endif

# By default we want to always auth to GCE to upload moby images
FORCE_GSUTIL_AUTH ?= 1

# Tag: 22b043cb43f6bc3bc4e1f1ecf9d64932f9325642
BIOS_IMAGE=mobylinux/mkimage-iso-bios@sha256:7e28f7745fd62284142dce59a137c3331f7673efe8b3f19ee5c65e4cc27ff18c

# Tag: 1edffcbfa13d4795f006d38e871a778ffba03d8a
PAD4_IMAGE=mobylinux/pad4@sha256:1ad26970698670373ee0bf374a06900f712a61b8038255e78271b840a1267b25

# Tag: d5711601eb5b89de0f052d87365e18388ff3f1b5
TAR2INITRD_IMAGE=mobylinux/tar2initrd@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

# Tag: 77bc577875fb3a80ac2d14b70d1daa885bbf199c
GCE_IMAGE=mobylinux/mkimage-gce@sha256:f9abf2eae20984b7dd3c1afb700b2c9c41e39e6e7c688c78348a51d0780d74cc

moby.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	$(MAKE) -C kernel
	$(MAKE) -j -C packages
	$(MAKE) -C base/ebpf
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
	docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp --tmpfs /initrd $$BUILD | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(PAD4_IMAGE) > $@

container.img:
	$(MAKE) -j -C containers
	tar cf - containers/*/rootfs containers/*/config.json | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

test.img:
	$(MAKE) -j -C test
	tar cf - test/rootfs test/config.json | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

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
		docker run --rm --net=none --log-driver=none -i $(BIOS_IMAGE) >$@

gce: gce.img.tar.gz

gce.img.tar.gz: common
	tar cf - initrd.img -C kernel/x86_64 vmlinuz64 | \
		docker run --rm --net=none --log-driver=none -i $(GCE_IMAGE) >$@

gce-upload:
ifeq ($(FORCE_GSUTIL_AUTH),1)
	-docker rm gsutil-moby-config 2>/dev/null
	docker run -it --name gsutil-moby-config google/cloud-sdk gcloud auth login --brief --project=docker-for-gcp
endif
	docker run --rm -ti \
		--volumes-from gsutil-moby-config \
		-v $(CURDIR)/gce.img.tar.gz:/gce.img.tar.gz \
		google/cloud-sdk \
		gsutil cp -a public-read /gce.img.tar.gz gs://docker-for-gcp-templates/latest/gce.img.tar.gz

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
	rm -f *.img *.vhd *.iso *.tag mobylinux.efi etc/moby-commit
	docker images -q moby-azure:build | xargs docker rmi -f || true
	docker images -q moby-azure:raw2vhd | xargs docker rmi -f || true
	docker volume rm vhdartifact || true
	$(MAKE) -C packages clean
	$(MAKE) -C containers clean
	$(MAKE) -C test clean
	$(MAKE) -C kernel clean
	$(MAKE) -C base/ebpf clean

.DELETE_ON_ERROR:
