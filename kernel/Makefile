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

RM = rm -f
# Name and Org on Hub
ORG?=linuxkit
IMAGE?=kernel
IMAGE_BUILDER=linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e

# You can specify an extra options for the Makefile. This will:
# - append a config$(EXTRA) to the kernel config for your kernel/arch
# - append $(EXTRA) to the CONFIG_LOCALVERSION of your kernel
EXTRA?=

DEBUG?=

ifeq ($(HASH),)
HASH_COMMIT?=HEAD # Setting this is only really useful with the show-tag target
HASH:=$(shell git ls-tree --full-tree $(HASH_COMMIT) -- $(CURDIR) | awk '{print $$3}')

ifneq ($(HASH_COMMIT),HEAD) # Others can't be dirty by definition
DIRTY:=$(shell git update-index -q --refresh && git diff-index --quiet HEAD -- $(CURDIR) || echo "-dirty")
endif
endif

# Path to push-manifest.sh
PUSH_MANIFEST:=$(shell git rev-parse --show-toplevel)/scripts/push-manifest.sh

# determine our architecture
BUILDERARCH=
ifneq ($(ARCH),)
ifeq ($(ARCH),$(filter $(ARCH),x86_64 amd64))
SUFFIX=-amd64
override ARCH=x86_64
BUILDERARCH=amd64
endif
ifeq ($(ARCH),$(filter $(ARCH),aarch64 arm64))
SUFFIX=-arm64
override ARCH=aarch64
BUILDERARCH=arm64
endif
endif

ifneq ($(BUILDERARCH),)
PLATFORMS=--platforms linux/$(BUILDERARCH)
endif

HASHTAG=$(HASH)$(DIRTY)

.PHONY: notdirty
notdirty:
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi

# utility function
SPACE := $(eval) $(eval)
PERIOD := .
series = $(word 1,$(subst ., ,$(1))).$(word 2,$(subst ., ,$(1))).x

# word 1 is the release, word 2 is the tool
RELEASESEP := PART
toolname = $(word 2, $(subst $(RELEASESEP), ,$(1)))
toolkernel = $(word 1, $(subst $(RELEASESEP), ,$(1)))
baseimageextension = :$(1)$(EXTRA)$(DEBUG)
baseimage = $(ORG)/$(IMAGE)$(call baseimageextension,$(1))
uniq = $(if $1,$(firstword $1) $(call uniq,$(filter-out $(firstword $1),$1)))


#
# Kernel versions to build.
# Use all for kernels to be built on all platforms; use KERNELS_x86_64 or KERNELS_aarch64 for platform-specific kernels
KERNELS_all=6.6.13 5.15.27
KERNELS_x86_64=
KERNELS_aarch64=

# deprecated versions. You might still be able to build them, but they are not built by default or supported
# Use all for kernels to be built on all platforms; use DEPRECATED_x86_64 or DEPRECATED_aarch64 for platform-specific kernels
DEPRECATED_all=5.10.104 5.11.4-rt
DEPRECATED_x86_64=5.4.172 
DEPRECATED_aarch64=

KERNELS?=$(KERNELS_all) $(KERNELS_$(ARCH))
DEPRECATED?=$(DEPRECATED_all) $(DEPRECATED_$(ARCH))

# we build all tools across all platforms and kernels that we build
TOOLS=bcc perf

# kernel versions used for kconfig
KERNEL_VERSIONS=$(call uniq,$(foreach l,$(KERNELS),$(word 1,$(subst -, ,$(l)))))

.PHONY: build push setforce show-tags list

list:
	@echo "Arch: $(ARCH)"
	@echo "Kernels: $(KERNELS)"
	@echo "Deprecated: $(DEPRECATED)"
	@echo "Tools: $(TOOLS)"

setforce:
	$(eval FORCE=--force)
build: $(addprefix build-,$(KERNELS))
push: $(addprefix push-,$(KERNELS))
show-tags: $(addprefix show-tag-,$(KERNELS))

build-%: buildkernel-% buildtools-%;

buildkernel-%: buildkerneldeps-% buildplainkernel-% builddebugkernel-%;

buildkerneldeps-%: Dockerfile Makefile $(wildcard patches-$(call series,$*)/*) $(wildcard config-$(call series,$*)*) ;

buildplainkernel-%: buildkerneldeps-%
	$(eval KERNEL_SERIES=$(call series,$*))
	linuxkit pkg build . $(FORCE) $(PLATFORMS) --build-yml ./build-kernel.yml --tag "$*-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args 

builddebugkernel-%: buildkerneldeps-%
	$(eval KERNEL_SERIES=$(call series,$*))
	linuxkit pkg build . $(FORCE) $(PLATFORMS) --build-yml ./build-kernel.yml --tag "$*-dbg-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args --build-arg-file build-args-debug

push-%: notdirty build-% pushkernel-% tagbuilder-% pushtools-%;

tagbuilder-%: notdirty
	$(eval BUILDER_IMAGE=$(call baseimage,$*)-builder)
	docker tag $(IMAGE_BUILDER) $(BUILDER_IMAGE)$(SUFFIX) && \
	docker push $(BUILDER_IMAGE)$(SUFFIX) && \
	$(PUSH_MANIFEST) $(BUILDER_IMAGE)

pushkernel-%: pushplainkernel-% pushdebugkernel-%;

pushplainkernel-%: buildplainkernel-%
	$(eval HASHED_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*-{{.Hash}}"))
	$(eval PLAIN_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*"))
	linuxkit cache push $(HASHED_IMAGE)
	linuxkit cache push $(HASHED_IMAGE) --remote-name $(PLAIN_IMAGE)

pushdebugkernel-%: builddebugkernel-%
	$(eval HASHED_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*-dbg-{{.Hash}}"))
	$(eval PLAIN_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*-dbg"))
	linuxkit cache push $(HASHED_IMAGE)
	linuxkit cache push $(HASHED_IMAGE) --remote-name $(PLAIN_IMAGE)

show-tag-%:
	@echo 	$(eval BASEIMAGE=$(call baseimage,$*))-$(HASHTAG)

buildtools-%: $(addprefix buildtool-%$(RELEASESEP),$(TOOLS));

buildtool-%:
	$(eval TOOL=$(call toolname,$*))
	$(eval KERNEL_VERSION=$(call toolkernel,$*))
	$(eval KERNEL_SERIES=$(call series,$(KERNEL_VERSION)))
	linuxkit pkg build . $(FORCE) $(PLATFORMS) --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args

pushtools-%: $(addprefix pushtool-%$(RELEASESEP),$(TOOLS));

pushtool-%: buildtool-%
	$(eval TOOL=$(call toolname,$*))
	$(eval KERNEL_VERSION=$(call toolkernel,$*))
	$(eval KERNEL_SERIES=$(call series,$(KERNEL_VERSION)))
	$(eval HASHED_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)-{{.Hash}}"))
	$(eval PLAIN_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)"))
	linuxkit cache push $(HASHED_IMAGE)
	linuxkit cache push $(HASHED_IMAGE) --remote-name $(PLAIN_IMAGE)

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
