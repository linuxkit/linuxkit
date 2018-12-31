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
IMAGE_BCC:=kernel-bcc
IMAGE_PERF:=kernel-perf
IMAGE_ZFS:=zfs-kmod

# You can specify an extra options for the Makefile. This will:
# - append a config$(EXTRA) to the kernel config for your kernel/arch
# - append $(EXTRA) to the CONFIG_LOCALVERSION of your kernel
EXTRA?=

# You can enable debug options for the Makefile. This will:
# - append a config-dbg to the kernel config for your kernel/arch
# - append -dbg to the CONFIG_LOCALVERSION of your kernel
DEBUG?=

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
ifeq ($(ARCH),x86_64)
SUFFIX=-amd64
endif
ifeq ($(ARCH),aarch64)
SUFFIX=-arm64
endif
ifeq ($(ARCH),s390x)
SUFFIX=-s390x
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

.PHONY: fetch build push
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
# $3: Build a specific kernel like -rt: Preempt-RT (used as suffix for image)
# This defines targets like:
# build_4.9.x and  push_4.9.x and adds them as dependencies
# to the global targets
# Set $3 to "-rt", to build Preempt-RT kernels. This defines targets like
# build_4.14.x-rt and adds "-rt" to the hub image name.
# Set $4 to "-dbg", to build debug kernels. This defines targets like
# build_4.9.x-dbg and adds "-dbg" to the hub image name.
# Set $3 to "-rt" and $4 to "-dbg" to build debug Preempt-RT kernel.
define kernel

ifeq ($(4),)
sources/linux-$(1).tar.xz: Makefile | sources
	curl -fsSLo sources/linux-$(1).tar.xz https://www.kernel.org/pub/linux/kernel/v4.x/linux-$(1).tar.xz 
KERNEL_VERSIONS+=$(1)
endif

build_$(2)$(3)$(4): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg | sources
	docker pull $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg EXTRA=$(3) \
			--build-arg DEBUG=$(4) \
			$(LABELS) \
			--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

forcebuild_$(2)$(3)$(4): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg | sources
	docker build \
		--build-arg KERNEL_VERSION=$(1) \
		--build-arg KERNEL_SERIES=$(2) \
		--build-arg EXTRA=$(3) \
		--build-arg DEBUG=$(4) \
		$(LABELS) \
		--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_$(2)$(3)$(4): build_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST))

forcepush_$(2)$(3)$(4): forcebuild_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	 docker tag $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
	 docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
	 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
	 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST)

show-tag_$(2)$(3)$(4):
	@echo $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)

build: build_$(2)$(3)$(4)
forcebuild: forcebuild_$(2)$(3)$(4)
push: push_$(2)$(3)$(4)
forcepush: forcepush_$(2)$(3)$(4)
show-tags: show-tag_$(2)$(3)$(4)
fetch: sources/linux-$(1).tar.xz

# 'docker build' with the FROM image supplied as --build-arg
# *and* with DOCKER_CONTENT_TRUST=1 currently does not work
# (https://github.com/moby/moby/issues/34199). So, we pull the image
# with DCT as part of the dependency on build_$(2)$(3)$(4) and then build
# with DOCKER_CONTENT_TRUST explicitly set to 0

ifeq ($(2), $(filter $(2),4.14.x 4.19.x))
# Only build perf for the latest LTS and the latest stable kernel.
ifneq ($(ARCH),aarch64)
# Skip arm64 for now as perf for 4.19.x fails to build.
build_perf_$(2)$(3)$(4): build_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.perf \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
			--no-cache --network=none $(LABEL) -t $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

forcebuild_perf_$(2)$(3)$(4): build_$(2)$(3)$(4)
	 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.perf \
		--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
		--no-cache --network=none $(LABEL) -t $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_perf_$(2)$(3)$(4): build_perf_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST))

forcepush_perf_$(2)$(3)$(4): forcebuild_perf_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
	docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST)

build: build_perf_$(2)$(3)$(4)
forcebuild: forcebuild_perf_$(2)$(3)$(4)
push: push_perf_$(2)$(3)$(4)
forcepush: forcepush_perf_$(2)$(3)$(4)
endif
endif

# For BCC, only build on x86 and only on latest LTS and latest stable.
# That's 4.14.x and 4.19.x at time of authorship.
ifeq ($(ARCH),x86_64)
ifeq ($(2), $(filter $(2),4.14.x 4.19.x))
build_bcc_$(2)$(3)$(4): build_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.bcc \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
			--no-cache $(LABEL) -t $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

forcebuild_bcc_$(2)$(3)$(4): build_$(2)$(3)$(4)
	 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.bcc \
		--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
		--no-cache $(LABEL) -t $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_bcc_$(2)$(3)$(4): build_bcc_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker pull $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST))

forcepush_bcc_$(2)$(3)$(4): forcebuild_bcc_$(2)$(3)$(4)
	@if [ x"$(DIRTY)" != x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi
	docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	docker tag $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
	docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG) $(DOCKER_CONTENT_TRUST) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4) $(DOCKER_CONTENT_TRUST)


build: build_bcc_$(2)$(3)$(4)
forcebuild: forcebuild_bcc_$(2)$(3)$(4)
push: push_bcc_$(2)$(3)$(4)
forcepush: forcepush_bcc_$(2)$(3)$(4)
endif
endif

ifeq ($(4),)
# ZFS does not compile against -dbg kernels because CONFIG_DEBUG_LOCK_ALLOC
# is incompatible with CDDL, apparently (this is ./configure check)
build_zfs_$(2)$(3): build_$(2)$(3)
	docker pull $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) || \
		 DOCKER_CONTENT_TRUST=0 docker build -f Dockerfile.zfs \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) \
			--no-cache $(LABEL) -t $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) .

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
ifeq ($(ARCH),x86_64)
$(eval $(call kernel,4.19.13,4.19.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.14.91,4.14.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.14.91,4.14.x,,-dbg))
$(eval $(call kernel,4.14.78,4.14.x,-rt,))
$(eval $(call kernel,4.9.148,4.9.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.4.169,4.4.x,$(EXTRA),$(DEBUG)))

else ifeq ($(ARCH),aarch64)
$(eval $(call kernel,4.19.13,4.19.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.14.91,4.14.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.14.78,4.14.x,-rt,))

else ifeq ($(ARCH),s390x)
$(eval $(call kernel,4.19.13,4.19.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,4.14.91,4.14.x,$(EXTRA),$(DEBUG)))
endif

# Target for kernel config
kconfig: | sources
	docker build --no-cache -f Dockerfile.kconfig \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		-t linuxkit/kconfig  .
