DEBUG ?= 0

all:	vmlinux push

# We push the image to hub twice, once with the full kernel version of
# "linuxkit/kernel:<kernel version>.<major version>.<minor version>-<n>",
# where "<n>" is a monotonically increasing config number, and as
# "linuxkit/kernel:<kernel version>.<major version>.x". This version
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
KERNEL_VERSION=4.9.22
IMAGE_VERSION=$(KERNEL_VERSION)-0
IMAGE_MAJOR_VERSION=4.9.x
DEPS=Dockerfile Makefile kernel_config kernel_config.debug patches-4.9
PKG_DEPS=bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar 

ifdef http_proxy
	BUILD_PROXY = --build-arg http_proxy=$(http_proxy)
	RUN_PROXY = --env http_proxy=$(http_proxy)
endif

ifdef https_proxy
	BUILD_PROXY += --build-arg https_proxy=$(https_proxy)
	RUN_PROXY += --env https_proxy=$(https_proxy)
endif

#build a kernel using dockerfile and save image hash in kernel.tag
kernel.tag: $(DEPS)
	BUILD=$$( tar cf - $^ | docker build -f $< $(BUILD_PROXY) --build-arg DEBUG=$(DEBUG) --build-arg KERNEL_VERSION=$(KERNEL_VERSION) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

#Extract $(PKG) from image with built kernel
vmlinux: kernel.tag
	mkdir -p x86_64
	docker run --rm --net=none --log-driver=none \
		$(shell cat kernel.tag)  \
		tar cf - $(PKG_DEPS) | tar xf - -C x86_64
	cp x86_64/kernel-modules.tar kernel.tar
	# FIXME: Remove when linuxkit allow get vmlinux from docker images
	# Rename vmlinux to bzImage
	cp x86_64/vmlinux bzImage
	cp x86_64/vmlinux $@

.PHONY: image push tag

MEDIA_TOYBOX=linuxkit/toybox-media:d7e82a7d19ccc84c9071fa7a88ecaa58ae958f7c@sha256:4c7d25f2be2429cd08417c36e04161cb924e46f3e419ee33a0aa9ff3a0942e02

BASE="$MEDIA_TOYBOX"
IMAGE=kernel-clear-containers

default: push

Dockerfile.media:
	printf "FROM $(MEDIA_TOYBOX)\nADD . /\n" > $@

image: Dockerfile.media vmlinux bzImage kernel.tar $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -f Dockerfile.media -

push: image
	docker pull linuxkit/$(IMAGE):$(IMAGE_VERSION) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(IMAGE_VERSION) && \
		 docker push linuxkit/$(IMAGE):$(IMAGE_VERSION) && \
		 docker tag $(IMAGE):build linuxkit/$(IMAGE):$(IMAGE_MAJOR_VERSION) && \
		 docker push linuxkit/$(IMAGE):$(IMAGE_MAJOR_VERSION))
	docker rmi $(IMAGE):build
	rm -f hash

tag: image
	(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(IMAGE_VERSION) && \
	docker tag $(IMAGE):build linuxkit/$(IMAGE):$(IMAGE_MAJOR_VERSION))
	docker rmi $(IMAGE):build
	rm -f hash

.PHONY: clean
clean:
	rm -rf x86_64 lib usr sbin kernel.tag Dockerfile.media vmlinux bzImage kernel.tar etc

.DELETE_ON_ERROR:
