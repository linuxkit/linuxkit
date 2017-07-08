# This builds the supported LinuxKit kernels. Kernels are wrapped up
# in a scratch container, which contains the bzImage, a tar
# ball with modules, the kernel sources, and in some case, the perf binary.
#
# Each kernel is pushed to hub twice:
# - linuxkit/kernel:<kernel>.<major>.<minor>-<hash>
# - linuxkit/kernel:<kernel>.<major>.<minor>
# The <hash> is the git tree hash of the current directory. The build
# will only rebuild the kernel image if the git tree hash changed.
#
# For some kernels we also build a separate package containing the perf utility
# which is specific to a given kernel. perf packages are tagged the same way
# kernel packages.

# Git tree hash of this directory. Override to force build
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
# Name and Org on Hub
ORG?=linuxkitprojects
IMAGE:=kernel-memorizer
IMAGE_PERF:=kernel-perf

# Add '-dirty' to hash if the repository is not clean. make does not
# concatenate strings without spaces, so we use the documented trick
# of replacing the space with nothing.
DIRTY=$(shell git diff-index --quiet HEAD --; echo $$?)
ifneq ($(DIRTY),0)
HASH+=-dirty
nullstring :=
space := $(nullstring) $(nullstring)
TAG=$(subst $(space),,$(HASH))
else
TAG=$(HASH)
endif

.PHONY: check tag push
# Targets:
# build: builds all kernels
# push:  pushes and sign all tagged kernel images to hub
build:
push:

# A template for defining kernel build
# Arguments:
# $1: Full kernel version, e.g., 4.9.22
# $2: Kernel "series", e.g., 4.9.x
# $3: Build a debug kernel (used as suffix for image)
# This defines targets like:
# build_4.9.x and  push_4.9.x and adds them as dependencies
# to the global targets
# Set $3 to "_dbg", to build debug kernels. This defines targets like
# build_4.9.x_dbg and adds "_dbg" to the hub image name.
define kernel
build_$(2)$(3): Dockerfile Makefile $(wildcard patches-$(2)/*) kernel_config-$(2) kernel_config.debug
	docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg DEBUG=$(3) \
			--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) .

push_$(2)$(3): build_$(2)$(3)
	@if [ $(DIRTY) -ne 0 ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) || \
		(DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) && \
		 docker tag $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) $(ORG)/$(IMAGE):$(1)$(3) && \
		 DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(1)$(3))

build: build_$(2)$(3)
push: push_$(2)$(3)


endef

#
# Build Targets
# Debug targets only for latest stable and LTS stable
#
#$(eval $(call kernel,4.10,4.10.x))
$(eval $(call kernel,4.10,4.10.x,_dbg))
#$(eval $(call kernel,4.11.7,4.11.x,_dbg))
#$(eval $(call kernel,4.9.34,4.9.x))
#$(eval $(call kernel,4.9.34,4.9.x,_dbg))
#$(eval $(call kernel,4.4.74,4.4.x))
