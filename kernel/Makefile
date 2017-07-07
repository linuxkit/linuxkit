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

# Name and Org on Hub
ORG?=linuxkit
IMAGE:=kernel
IMAGE_PERF:=kernel-perf

ifeq ($(HASH),)
HASH_COMMIT?=HEAD # Setting this is only really useful with the show-tag target
HASH?=$(shell git ls-tree --full-tree $(HASH_COMMIT) -- $(CURDIR) | awk '{print $$3}')

ifneq ($(HASH_COMMIT),HEAD) # Others can't be dirty by definition
DIRTY=$(shell  git update-index -q --refresh && git diff-index --quiet HEAD -- $(CURDIR) || echo "-dirty")
endif
endif

TAG=$(HASH)$(DIRTY)

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
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) || \
		(DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) && \
		 docker tag $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) $(ORG)/$(IMAGE):$(1)$(3) && \
		 DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(1)$(3))

show-tag_$(2)$(3):
	@echo $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)

build: build_$(2)$(3)
push: push_$(2)$(3)
show-tags: show-tag_$(2)$(3)

ifneq ($(2), 4.4.x)
build_perf_$(2)$(3): build_$(2)$(3)
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) || \
		docker build -f Dockerfile.perf \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)-$(TAG) \
	 		--no-cache --network=none -t $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) .

push_perf_$(2)$(3): build_perf_$(2)$(3)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) || \
		(DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) && \
		 docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) $(ORG)/$(IMAGE_PERF):$(1)$(3) && \
		 DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE_PERF):$(1)$(3))

build: build_perf_$(2)$(3)
push: push_perf_$(2)$(3)
endif

endef

#
# Build Targets
# Debug targets only for latest stable and LTS stable
#
$(eval $(call kernel,4.11.9,4.11.x))
$(eval $(call kernel,4.11.9,4.11.x,_dbg))
$(eval $(call kernel,4.9.36,4.9.x))
$(eval $(call kernel,4.9.36,4.9.x,_dbg))
$(eval $(call kernel,4.4.76,4.4.x))
