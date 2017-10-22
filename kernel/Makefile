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
IMAGE_ZFS:=zfs-kmod

# You can specify an extra options for the Makefile. This will:
# - append a config$(EXTRA) to the kernel config for your kernel/arch
# - append $(EXTRA) to the CONFIG_LOCALVERSION of your kernel
EXTRA?=

ifeq ($(HASH),)
HASH_COMMIT?=HEAD # Setting this is only really useful with the show-tag target
HASH?=$(shell git ls-tree --full-tree $(HASH_COMMIT) -- $(CURDIR) | awk '{print $$3}')

ifneq ($(HASH_COMMIT),HEAD) # Others can't be dirty by definition
DIRTY=$(shell  git update-index -q --refresh && git diff-index --quiet HEAD -- $(CURDIR) || echo "-dirty")
endif
endif

# Path to push-manifest.sh
PUSH_MANIFEST:=$(shell git rev-parse --show-toplevel)/scripts/push-manifest.sh

ARCH := $(shell uname -m)
ifeq ($(ARCH), x86_64)
SUFFIX=-amd64
endif
ifeq ($(ARCH), aarch64)
SUFFIX=-arm64
endif

TAG=$(HASH)$(DIRTY)

REPO?=https://github.com/linuxkit/linuxkit
ifneq ($(REPO),)
REPO_LABEL=--label org.opencontainers.image.source=$(REPO)
endif
ifeq ($(DIRTY),)
REPO_COMMIT=$(shell git rev-parse HEAD)
COMMIT_LABEL=--label org.opencontainers.image.revision=$(REPO_COMMIT)
endif
LABELS=$(REPO_LABEL) $(COMMIT_LABEL)

ifeq ($(DOCKER_CONTENT_TRUST),)
ifndef NOTRUST
export DOCKER_CONTENT_TRUST=1
endif
endif

KERNEL_VERSIONS=

.PHONY: check build push
# Targets:
# fetch: Downloads the kernel sources into ./sources
# build: Builds all kernels
# push:  Pushes and sign all tagged kernel images to hub
fetch:
build:
push:

sources:
	mkdir -p $@

# A template for defining kernel build
# Arguments:
# $1: Full kernel version, e.g., 4.9.22
# $2: Kernel "series", e.g., 4.9.x
# $3: Build a debug kernel (used as suffix for image)
# This defines targets like:
# build_4.9.x and  push_4.9.x and adds them as dependencies
# to the global targets
# Set $3 to "-dbg", to build debug kernels. This defines targets like
# build_4.9.x-dbg and adds "-dbg" to the hub image name.
define kernel

ifeq ($(3),)
sources/linux-$(1).tar.xz: Makefile | sources
	curl -fsSLo sources/linux-$(1).tar.xz https://www.kernel.org/pub/linux/kernel/v4.x/linux-$(1).tar.xz 
KERNEL_VERSIONS+=$(1)
endif

build_$(2)$(3): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg | sources
	docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg EXTRA=$(3) \
			$(LABELS) \
			--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) .

push_$(2)$(3): build_$(2)$(3)
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE):$(1)$(3)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE):$(1)$(3)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3) $(DOCKER_CONTENT_TRUST))

show-tag_$(2)$(3):
	@echo $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)

build: build_$(2)$(3)
push: push_$(2)$(3)
show-tags: show-tag_$(2)$(3)
fetch: sources/linux-$(1).tar.xz

# 'docker build' with the FROM image supplied as --build-arg
# *and* with DOCKER_CONTENT_TRUST=1 currently does not work
# (https://github.com/moby/moby/issues/34199). So, we pull the image
# with DCT and then build with DOCKER_CONTENT_TRUST explicitly set to 0.

ifneq ($(2), 4.4.x)
# perf does not build out of the box for 4.4.x and 4.4.x is not that relevant anymore to work on a fix
build_perf_$(2)$(3): build_$(2)$(3)
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.perf \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) \
			--no-cache --network=none $(LABEL) -t $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG)$(SUFFIX) .)

push_perf_$(2)$(3): build_perf_$(2)$(3)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3) $(DOCKER_CONTENT_TRUST))

build: build_perf_$(2)$(3)
push: push_perf_$(2)$(3)
endif

ifneq ($(3), -dbg)
# ZFS does not compile against -dbg kernels because CONFIG_DEBUG_LOCK_ALLOC
# is incompatible with CDDL, apparently (this is ./configure check)
build_zfs_$(2)$(3): build_$(2)$(3)
	docker pull $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker pull $(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.zfs \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) \
			--no-cache $(LABEL) -t $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) .)

push_zfs_$(2)$(3): build_zfs_$(2)$(3)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_ZFS):$(1)$(3)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_ZFS):$(1)$(3)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_ZFS):$(1)$(3) $(DOCKER_CONTENT_TRUST))
endif

endef

#
# Build Targets
# Debug targets only for latest stable and LTS stable
#
$(eval $(call kernel,4.13.9,4.13.x,$(EXTRA)))
$(eval $(call kernel,4.13.9,4.13.x,-dbg))
$(eval $(call kernel,4.9.58,4.9.x,$(EXTRA)))
$(eval $(call kernel,4.9.58,4.9.x,-dbg))
$(eval $(call kernel,4.4.94,4.4.x,$(EXTRA)))

# Target for kernel config
kconfig: | sources
	docker build --no-cache -f Dockerfile.kconfig \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		-t linuxkit/kconfig  .
