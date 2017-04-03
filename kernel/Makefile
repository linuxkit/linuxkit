DEBUG ?= 0

all:	bzImage tag

# We push the image to hub twice, once with the full kernel version of
# "mobylinux/kernel:<kernel version>.<major version>.<minor version>-<n>",
# where "<n>" is a monotonically increasing config number, and as
# "mobylinux/kernel:<kernel version>.<major version>.x". This version
# number is stored in IMAGE_VERSION.
#
# We expect most users to us the "<kernel version>.<major version>.x"
# variant as this simply is the latest version of a given major kernel
# version. This version number is stored in IMAGE_MAJOR_VERSION.
#
# For IMAGE_VERSION, the "<n>" must be increased whenever
# the kernel config or the patches change. We don't expect this to
# happen very often as the minor version number gets update quite
# frequently.
#
# IMAGE_VERSION is used to determine if a new image should be pushed to hub.
ifeq ($(KERNEL),v4.4)
KERNEL_VERSION=4.4.59
IMAGE_VERSION=$(KERNEL_VERSION)-0
IMAGE_MAJOR_VERSION=4.4.x
DEPS=Dockerfile.4.4 Makefile kernel_config kernel_config.debug kernel_config.4.4 patches-4.4
else
ifeq ($(KERNEL),v4.10)
KERNEL_VERSION=4.10.8
IMAGE_VERSION=$(KERNEL_VERSION)-0
IMAGE_MAJOR_VERSION=4.10.x
DEPS=Dockerfile.4.10 Makefile kernel_config kernel_config.debug patches-4.10
else
KERNEL_VERSION=4.9.20
IMAGE_VERSION=$(KERNEL_VERSION)-0
IMAGE_MAJOR_VERSION=4.9.x
DEPS=Dockerfile Makefile kernel_config kernel_config.debug patches-4.9
endif
endif

kernel.tag: $(DEPS)
	BUILD=$$( tar cf - $^ | docker build -f $< --build-arg DEBUG=$(DEBUG) --build-arg KERNEL_VERSION=$(KERNEL_VERSION) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

bzImage: kernel.tag
	rm -rf etc/kernel-patches
	mkdir -p x86_64 etc lib usr sbin etc/kernel-patches
	docker run --rm --net=none --log-driver=none $(shell cat kernel.tag) tar cf - bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar | tar xf - -C x86_64
	cp x86_64/kernel-modules.tar kernel.tar
	cp x86_64/bzImage $@

.PHONY: image push tag

MEDIA_TOYBOX=mobylinux/toybox-media:0a26fe5f574e444849983f9c4148ef74b3804d55@sha256:5ac38f77b66deb194c9016591b9b096e81fcdc9f7c3e6d01566294a6b4b4ebd2

BASE="$MEDIA_TOYBOX"
IMAGE=kernel

default: push

Dockerfile.media:
	printf "FROM $(MEDIA_TOYBOX)\nADD . /\n" > $@

image: Dockerfile.media bzImage kernel.tar $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -f Dockerfile.media -

push: image
	docker pull mobylinux/$(IMAGE):$(IMAGE_VERSION) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(IMAGE_VERSION) && \
		 docker push mobylinux/$(IMAGE):$(IMAGE_VERSION) && \
		 docker tag $(IMAGE):build mobylinux/$(IMAGE):$(IMAGE_MAJOR_VERSION) && \
		 docker push mobylinux/$(IMAGE):$(IMAGE_MAJOR_VERSION))
	docker rmi $(IMAGE):build
	rm -f hash

tag: image
	(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(IMAGE_VERSION) && \
	docker tag $(IMAGE):build mobylinux/$(IMAGE):$(IMAGE_MAJOR_VERSION))
	docker rmi $(IMAGE):build
	rm -f hash

.PHONY: clean
clean:
	rm -rf x86_64 lib usr sbin kernel.tag Dockerfile.media bzImage kernel.tar

.DELETE_ON_ERROR:
