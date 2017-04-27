# This builds the supported LinuxKit kernels. Kernels are wrapped up
# in a minimal toybox container, which contains the bzImage, a tar
# ball with modules and the kernel source.
#
# Each kernel is pushed to hub twice, once as
# linuxkit/kernel:<kernel>.<major>.<minor>-<hash> and once as
# inuxkit/kernel:<kernel>.<major>.x. The <hash> is the git tree hash
# of the current directory. The build will only rebuild the kernel
# image if the git tree hash changed.

# Enable a debug kernel
DEBUG?=0
# Git tree hash of this directory. Override to force build
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
# Name on Hub
IMAGE:=kernel

.PHONY: check tag push sign
# Targets:
# build: builds all kernels
# push:  pushes all tagged kernel images to hub
# sign:  sign and push all kernel images to hub
build:
push:
sign:

# A template for defining kernel build
# Arguments:
# $1: Full kernel version, e.g., 4.9.22
# $2: Kernel "series", e.g., 4.9.x
# This defines targets like:
# tag_4.9.x, push_4.9.x and sign_4.9.x
# and adds them as dependencies to the global targets
define kernel
build_$(2): Dockerfile Makefile $(wildcard patches-$(2)/*) kernel_config-$(2) kernel_config.debug
	docker pull linuxkit/$(IMAGE):$(1)-$(HASH) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--no-cache -t linuxkit/$(IMAGE):$(1)-$(HASH) .

push_$(2): build_$(2)
	docker pull linuxkit/$(IMAGE):$(1)-$(HASH) || \
		(docker push linuxkit/$(IMAGE):$(1)-$(HASH) && \
		 docker tag linuxkit/$(IMAGE):$(1)-$(HASH) linuxkit/$(IMAGE):$(2) && \
		 docker push linuxkit/$(IMAGE):$(2))

sign_$(2): build_$(2)
	DOCKER_CONTENT_TRUST=1 docker pull linuxkit/$(IMAGE):$(1)-$(HASH) || \
		(DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(1)-$(HASH) && \
		 docker tag linuxkit/$(IMAGE):$(1)-$(HASH) linuxkit/$(IMAGE):$(2) && \
		 DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(2))

build: build_$(2)
push: push_$(2)
sign: sign_$(2)
endef

#
# Build Targets
#
$(eval $(call kernel,4.10.12,4.10.x))
$(eval $(call kernel,4.9.24,4.9.x))
$(eval $(call kernel,4.4.63,4.4.x))
