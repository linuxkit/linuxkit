# This builds the supported LinuxKit kernels. Kernels are wrapped up
# in a minimal toybox container, which contains the bzImage, a tar
# ball with modules and the kernel source.
#
# Each kernel is pushed to hub twice, once as
# linuxkit/kernel:<kernel>.<major>.<minor>-<hash> and once as
# inuxkit/kernel:<kernel>.<major>.x. The <hash> is the git tree hash
# of the current directory. The build will only rebuild the kernel
# image if the git tree hash changed.

# Git tree hash of this directory. Override to force build
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
# Name on Hub
IMAGE:=kernel
ARCH?=x86_64

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
# $3: Build a debug kernel (used as suffix for image)
# This defines targets like:
# build_4.9.x, push_4.9.x and sign_4.9.x and adds them as dependencies
# to the global targets
# Set $3 to "_dbg", to build debug kernels. This defines targets like
# build_4.9.x_dbg and adds "_dbg" to the hub image name.
define kernel
build_$(2)$(3): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard kernel_config.$(2)*) kernel_config.base kernel_config.$(ARCH)
	cp ../../test/pkg/kernel-config/check-kernel-config.sh .
	docker pull linuxkit/$(IMAGE):$(1)$(3)-$(HASH) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg ARCH=$(ARCH) \
			--build-arg DEBUG=$(3) \
			--no-cache -t linuxkit/$(IMAGE):$(1)$(3)-$(HASH) .
	-rm check-kernel-config.sh

push_$(2)$(3): build_$(2)$(3)
	docker pull linuxkit/$(IMAGE):$(1)$(3)-$(HASH) || \
		(docker push linuxkit/$(IMAGE):$(1)$(3)-$(HASH) && \
		 docker tag linuxkit/$(IMAGE):$(1)$(3)-$(HASH) linuxkit/$(IMAGE):$(2)$(3) && \
		 docker push linuxkit/$(IMAGE):$(2)$(3))

sign_$(2)$(3): build_$(2)$(3)
	DOCKER_CONTENT_TRUST=1 docker pull linuxkit/$(IMAGE):$(1)$(3)-$(HASH) || \
		(DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(1)$(3)-$(HASH) && \
		 docker tag linuxkit/$(IMAGE):$(1)$(3)-$(HASH) linuxkit/$(IMAGE):$(2)$(3) && \
		 DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(2)$(3))

build: build_$(2)$(3)
push: push_$(2)$(3)
sign: sign_$(2)$(3)
endef

#
# Build Targets
# Debug targets only for latest stable and LTS stable
#
$(eval $(call kernel,4.11.2,4.11.x))
$(eval $(call kernel,4.11.2,4.11.x,_dbg))
$(eval $(call kernel,4.10.17,4.10.x))
$(eval $(call kernel,4.10.17,4.10.x,_dbg))
$(eval $(call kernel,4.9.29,4.9.x))
$(eval $(call kernel,4.9.29,4.9.x,_dbg))
