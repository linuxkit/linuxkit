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
PLATFORMS?=linux/amd64,linux/arm64
IMAGE?=kernel
IMAGE_BCC:=kernel-bcc
IMAGE_PERF:=kernel-perf
IMAGE_ZFS:=zfs-kmod
IMAGE_BUILDER=linuxkit/alpine:2be490394653b7967c250e86fd42cef88de428ba

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
ifeq ($(ARCH),$(filter $(ARCH),aarch64 arm64))
SUFFIX=-arm64
endif

TAG=$(HASH)$(DIRTY)

BUILD_LABEL=--label org.mobyproject.linuxkit.kernel.buildimage=$(IMAGE_BUILDER)

REPO?=https://github.com/linuxkit/linuxkit
ifneq ($(REPO),)
REPO_LABEL=--label org.opencontainers.image.source=$(REPO)
endif
ifeq ($(DIRTY),)
REPO_COMMIT=$(shell git rev-parse HEAD)
COMMIT_LABEL=--label org.opencontainers.image.revision=$(REPO_COMMIT)
endif

LABELS=$(REPO_LABEL) $(COMMIT_LABEL) $(BUILD_LABEL)

KERNEL_VERSIONS=

.PHONY: build push
# Targets:
# build: Builds all kernels
# push:  Pushes and sign all tagged kernel images to hub
build:
push:

.PHONY: notdirty
notdirty:
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi


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
KERNEL_VERSIONS+=$(1)
endif

buildx_$(2)$(3)$(4): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg
	docker pull $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) || \
		docker buildx build \
			--platform=$(PLATFORMS) --push \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg EXTRA=$(3) \
			--build-arg DEBUG=$(4) \
			--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
			$(LABELS) \
			--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) .

build_$(2)$(3)$(4): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg
	docker pull $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		docker build \
			--build-arg KERNEL_VERSION=$(1) \
			--build-arg KERNEL_SERIES=$(2) \
			--build-arg EXTRA=$(3) \
			--build-arg DEBUG=$(4) \
			--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
			$(LABELS) \
			--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .


forcebuild_$(2)$(3)$(4): Dockerfile Makefile $(wildcard patches-$(2)/*) $(wildcard config-$(2)*) config-dbg
	docker build \
		--build-arg KERNEL_VERSION=$(1) \
		--build-arg KERNEL_SERIES=$(2) \
		--build-arg EXTRA=$(3) \
		--build-arg DEBUG=$(4) \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		$(LABELS) \
		--no-cache -t $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_$(2)$(3)$(4): notdirty build_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4))

forcepush_$(2)$(3)$(4): notdirty forcebuild_$(2)$(3)$(4)
	docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	 docker tag $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
	 docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)$(SUFFIX) && \
	 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG) && \
	 $(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)

# tag the builder and create the manifest
tagbuilder_$(2)$(3)$(4): notdirty
	docker tag $(IMAGE_BUILDER) $(ORG)/$(IMAGE):$(1)$(3)$(4)-builder$(SUFFIX) && \
	docker push $(ORG)/$(IMAGE):$(1)$(3)$(4)-builder$(SUFFIX) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE):$(1)$(3)$(4)-builder


show-tag_$(2)$(3)$(4):
	@echo $(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)

build: build_$(2)$(3)$(4)
forcebuild: forcebuild_$(2)$(3)$(4)
push: push_image tagbuilder
push_image: push_$(2)$(3)$(4)
forcepush: forcepush_image tagbuilder
forcepush_image: forcepush_$(2)$(3)$(4)
tagbuilder: tagbuilder_$(2)$(3)$(4)
show-tags: show-tag_$(2)$(3)$(4)

# Only build perf only on x86 and recent LTS and latest stable kernels
ifeq ($(ARCH),x86_64)
ifeq ($(2), $(filter $(2),5.15.x 5.10.x 5.4.x))
build_perf_$(2)$(3)$(4): build_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		 docker build -f Dockerfile.perf \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
			--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
			--no-cache --network=none $(LABEL) -t $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

forcebuild_perf_$(2)$(3)$(4): build_$(2)$(3)$(4)
	docker build -f Dockerfile.perf \
		--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		--no-cache --network=none $(LABEL) -t $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_perf_$(2)$(3)$(4): notdirty build_perf_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4))

forcepush_perf_$(2)$(3)$(4): notdirty forcebuild_perf_$(2)$(3)$(4)
	docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	docker tag $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
	docker push $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)$(SUFFIX) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)-$(TAG) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_PERF):$(1)$(3)$(4)

build: build_perf_$(2)$(3)$(4)
forcebuild: forcebuild_perf_$(2)$(3)$(4)
push: push_perf_$(2)$(3)$(4)
forcepush: forcepush_perf_$(2)$(3)$(4)
endif
endif

# Only build bcc only on x86 and recent LTS and latest stable kernels
ifeq ($(ARCH),x86_64)
ifeq ($(2), $(filter $(2),5.15.x 5.10.x 5.4.x))
build_bcc_$(2)$(3)$(4): build_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		docker build -f Dockerfile.bcc \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
			--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
			--no-cache $(LABEL) -t $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

forcebuild_bcc_$(2)$(3)$(4): build_$(2)$(3)$(4)
	 docker build -f Dockerfile.bcc \
		--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)$(4)-$(TAG)$(SUFFIX) \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		--no-cache $(LABEL) -t $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) .

push_bcc_$(2)$(3)$(4): notdirty build_bcc_$(2)$(3)$(4)
	docker pull $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4))

forcepush_bcc_$(2)$(3)$(4): notdirty forcebuild_bcc_$(2)$(3)$(4)
	docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) && \
	docker tag $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
	docker push $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)$(SUFFIX) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)-$(TAG) && \
	$(PUSH_MANIFEST) $(ORG)/$(IMAGE_BCC):$(1)$(3)$(4)

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
		docker build -f Dockerfile.zfs \
			--build-arg IMAGE=$(ORG)/$(IMAGE):$(1)$(3)-$(TAG)$(SUFFIX) \
			--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
			--no-cache $(LABEL) -t $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) .

push_zfs_$(2)$(3): notdirty build_zfs_$(2)$(3)
	docker pull $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) || \
		(docker push $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) && \
		 docker tag $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG)$(SUFFIX) $(ORG)/$(IMAGE_ZFS):$(1)$(3)$(SUFFIX) && \
		 docker push $(ORG)/$(IMAGE_ZFS):$(1)$(3)$(SUFFIX) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_ZFS):$(1)$(3)-$(TAG) && \
		 $(PUSH_MANIFEST) $(ORG)/$(IMAGE_ZFS):$(1)$(3))
endif

endef

#
# Build Targets
# Debug targets only for latest stable and LTS stable
#
ifeq ($(ARCH),x86_64)
$(eval $(call kernel,5.15.27,5.15.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,5.15.27,5.15.x,,-dbg))
$(eval $(call kernel,5.10.104,5.10.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,5.4.172,5.4.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,5.11.4,5.11.x,-rt,))

else ifeq ($(ARCH),$(filter $(ARCH),aarch64 arm64))
$(eval $(call kernel,5.15.27,5.15.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,5.10.104,5.10.x,$(EXTRA),$(DEBUG)))
$(eval $(call kernel,5.11.4,5.11.x,-rt,))

endif

# Target for kernel config
kconfig:
ifeq (${KCONFIG_TAG},)
	docker build --no-cache -f Dockerfile.kconfig \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfig  .
else
	docker build --no-cache -f Dockerfile.kconfig \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfig:${KCONFIG_TAG}  .
endif

kconfigx:
ifeq (${KCONFIG_TAG},)
	docker buildx build --no-cache -f Dockerfile.kconfigx \
		--platform=$(PLATFORMS) \
		--output . \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfigx  .
	cp linux_arm64/config-${KERNEL_VERSIONS}-arm64 config-${KERNEL_SERIES}-aarch64
	cp linux_amd64/config-${KERNEL_VERSIONS}-amd64 config-${KERNEL_SERIES}-x86_64
else
	docker buildx build --no-cache -f Dockerfile.kconfigx \
		--platform=$(PLATFORMS) --push \
		--output . \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfigx:${KCONFIG_TAG}  .
endif
