all:	initrd.img initrd-test.img mobylinux-efi.iso mobylinux-bios.iso ebpf

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

# Tag: 2a860edda12a44c8e141a924f29ea931dbc01110
BIOS_IMAGE=mobylinux/mkimage-iso-bios@sha256:974304cfff80524a37bf96d299de9b3163ba07df5feeb620c6bdcd132b46f16d

# Tag: f81034d118744a42e8e93cfe0777dc3490c1f6a0
EFI_IMAGE=mobylinux/mkimage-iso-efi@sha256:4bb10d794fd0fe58366c36ab0fabf7b6d22e5bd760a994900da3b910230519cc

# Tag: 1edffcbfa13d4795f006d38e871a778ffba03d8a
PAD4_IMAGE=mobylinux/pad4@sha256:1ad26970698670373ee0bf374a06900f712a61b8038255e78271b840a1267b25

# Tag: d5711601eb5b89de0f052d87365e18388ff3f1b5
TAR2INITRD_IMAGE=mobylinux/tar2initrd@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

# Tag: d56cde1558e3080e59a32e3cd7c7141baa601811
TARTAR2INITRD_IMAGE=mobylinux/tartar2initrd@sha256:e1ad4522ff906d339da5f250b9ef6bffa5a70b4dec7d2cf7f7dbd0447b79352f

# Tag: 77bc577875fb3a80ac2d14b70d1daa885bbf199c
GCE_IMAGE=mobylinux/mkimage-gce@sha256:f9abf2eae20984b7dd3c1afb700b2c9c41e39e6e7c688c78348a51d0780d74cc

moby.img: Dockerfile mkinitrd.sh init $(ETCFILES)
	$(MAKE) -j -C packages
	printf $(TAG)$(DIRTY) > etc/moby-commit
	BUILD=$$( tar cf - \
	  Dockerfile etc usr init mkinitrd.sh \
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
	  -C packages/containerd etc usr -C ../.. \
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
	tar cf - $$(find containers -name container.tar) | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TARTAR2INITRD_IMAGE) > $@

test.img:
	$(MAKE) -j -C test
	cat test/container.tar | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

kernel/x86_64/kernel.img:
	$(MAKE) -C kernel

initrd.img: moby.img kernel/x86_64/kernel.img container.img
	cat $^ > $@

initrd-test.img: initrd.img test.img
	cat $^ > $@

.PHONY: epbf
ebpf: kernel/x86_64/kernel.img
	$(MAKE) -C ebpf

# outputs tarball of mobylinux-efi.iso mobylinux.efi
mobylinux-efi.iso: initrd.img kernel/x86_64/vmlinuz64
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(EFI_IMAGE) | tar xf -

mobylinux-bios.iso: initrd.img kernel/x86_64/vmlinuz64
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(BIOS_IMAGE) >$@

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
	rm -f *.img *.vhd *.iso *.tag mobylinux.efi etc/moby-commit gce.img.tar.gz
	docker images -q moby-azure:build | xargs docker rmi -f || true
	docker images -q moby-azure:raw2vhd | xargs docker rmi -f || true
	docker volume rm vhdartifact || true
	$(MAKE) -C packages clean
	$(MAKE) -C containers clean
	$(MAKE) -C test clean
	$(MAKE) -C kernel clean
	$(MAKE) -C ebpf clean

.DELETE_ON_ERROR:
