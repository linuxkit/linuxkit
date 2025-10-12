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
IMAGE_BUILDER=linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a

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

REPO_ROOT:=$(shell git rev-parse --show-toplevel)

# determine our architecture
ARCH?=$(shell uname -m)
BUILDERARCH=$(ARCH)
ifneq ($(ARCH),)
ifeq ($(ARCH),$(filter $(ARCH),x86_64 amd64))
override ARCH=x86_64
override BUILDERARCH=amd64
endif
ifeq ($(ARCH),$(filter $(ARCH),aarch64 arm64))
override ARCH=aarch64
override BUILDERARCH=arm64
endif
ifeq ($(ARCH),riscv64)
override BUILDERARCH=riscv64
endif
endif

BUILD_PLATFORM=linux/$(BUILDERARCH)

HASHTAG=$(HASH)$(DIRTY)

.PHONY: notdirty
notdirty:
	@if [ x"$(DIRTY)" !=  x ]; then echo "Your repository is not clean. Will not push image"; exit 1; fi

# utility function
SPACE := $(eval) $(eval)
PERIOD := .
# series - convert a version to a series, e.g. 6.6.13 -> 6.6.x
series = $(word 1,$(subst ., ,$(1))).$(word 2,$(subst ., ,$(1))).x
# serieswithhash - convert a version with or without a hash to a series with a hash, e.g. 6.6.13-anbcd -> 6.6.x-[0-9a-f]+
serieswithhash = $(word 1,$(subst ., ,$(1))).$(word 2,$(subst ., ,$(1))).[0-9]+-[0-9a-f]+

# word 1 is the release, word 2 is the tool
RELEASESEP := PART
toolname = $(word 2, $(subst $(RELEASESEP), ,$(1)))
toolkernel = $(word 1, $(subst $(RELEASESEP), ,$(1)))
baseimageextension = :$(1)$(EXTRA)$(DEBUG)
baseimage = $(ORG)/$(IMAGE)$(call baseimageextension,$(1))
uniq = $(if $1,$(firstword $1) $(call uniq,$(filter-out $(firstword $1),$1)))


# DEPRECATED : all kernel versions (actually series) marked as deprecated
# You might still be able to build them, but they are not built by default or supported
DEPRECATED_list=$(wildcard */deprecated)
DEPRECATED := $(patsubst %/deprecated,%,$(DEPRECATED_list))
#
# KERNELS : all potential kernel versions, based on the build-args files

# first find all known build-args files
KERNELS_buildargfiles=$(wildcard */build-args)
# get their directories
KERNELS_alldirs=$(patsubst %/build-args,%,$(KERNELS_buildargfiles))
# remove any directories that are marked as deprecated; what is left is valid dirs
KERNELS_validdirs=$(filter-out $(DEPRECATED),$(KERNELS_alldirs))
# get the values from the valid dirs
KERNELS=$(shell awk -F= '/^KERNEL_VERSION=/ {print $$2}' $(addsuffix /build-args,$(KERNELS_validdirs)))

# get the highest supported one
KERNEL_HIGHEST=$(shell echo $(KERNELS) | tr ' ' '\n' | sort -V | tail -n 1)


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
	linuxkit pkg build . $(FORCE) --platforms $(BUILD_PLATFORM) --build-yml ./build-kernel.yml --tag "$*-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args 

builddebugkernel-%: buildkerneldeps-%
	$(eval KERNEL_SERIES=$(call series,$*))
	linuxkit pkg build . $(FORCE) --platforms $(BUILD_PLATFORM) --build-yml ./build-kernel.yml --tag "$*-dbg-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args --build-arg-file build-args-debug

push-%: notdirty build-% pushkernel-% tagbuilder-% pushtools-%;

# tagbuilder-% tags the builder image with the kernel version and `-builder` and pushes it
# checks if it already matches on the registry before pushing
# because the build may have been on a remote builder, or we may not have had to do a local build,
# we cannot assume that IMAGE_BUILDER is available locally, whether in docker image cache or limuxkit cache
tagbuilder-%: notdirty
	$(eval BUILDER_IMAGE=$(call baseimage,$*)-builder)
	linuxkit pkg remote-tag $(IMAGE_BUILDER) $(BUILDER_IMAGE)

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
	@echo 	$(call baseimage,$*)-$(HASHTAG)

buildtools-%: $(addprefix buildtool-%$(RELEASESEP),$(TOOLS));

buildtool-%:
	$(eval TOOL=$(call toolname,$*))
	$(eval KERNEL_VERSION=$(call toolkernel,$*))
	$(eval KERNEL_SERIES=$(call series,$(KERNEL_VERSION)))
	linuxkit pkg build . $(FORCE) --platforms $(BUILD_PLATFORM) --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)-{{.Hash}}" --build-arg-file $(KERNEL_SERIES)/build-args

pushtools-%: $(addprefix pushtool-%$(RELEASESEP),$(TOOLS));

pushtool-%: buildtool-%
	$(eval TOOL=$(call toolname,$*))
	$(eval KERNEL_VERSION=$(call toolkernel,$*))
	$(eval KERNEL_SERIES=$(call series,$(KERNEL_VERSION)))
	$(eval HASHED_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)-{{.Hash}}"))
	$(eval PLAIN_IMAGE=$(shell linuxkit pkg show-tag . --build-yml ./build-$(TOOL).yml --tag "$(KERNEL_VERSION)"))
	linuxkit cache push $(HASHED_IMAGE)
	linuxkit cache push $(HASHED_IMAGE) --remote-name $(PLAIN_IMAGE)

#
# targets for getting names of particular tags and replacing them, like what scripts/update-component-sha.sh does
#

# get the tag for the normal kernel for a particular version. Accepts version or series
tag-plainkernel-%:
	@linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*-{{.Hash}}"

# get the tag for the debug kernel for a particular version. Accepts version or series
tag-debugkernel-%:
	@linuxkit pkg show-tag . --build-yml ./build-kernel.yml --tag "$*-dbg-{{.Hash}}"

# find and replace any usage of the normal kernel with hash for a particular series
# will update hash for same semver and/or patch version
update-kernel-hash-yaml-%:
	$(eval NEWTAG=$(shell $(MAKE) tag-plainkernel-$*))
	$(eval OLDTAG=$(call serieswithhash,$(NEWTAG)))
	@cd $(REPO_ROOT) && ./scripts/update-component-sha.sh --hash "$(OLDTAG)" "$(NEWTAG)"

# find and replace any usage of the normal kernel with semver for most recent series
update-kernel-semver-yaml-%:
	$(eval NEWTAG=linuxkit/kernel:$*)
	$(eval OLDTAG=linuxkit/kernel:[0-9]+.[0-9]+.[0-9]+)
	@cd $(REPO_ROOT) && ./scripts/update-component-sha.sh --hash "$(OLDTAG)" "$(NEWTAG)"

# update-kernel-yamls updates the latest hash for each supported series,
# as well as the most recent supported semver
update-kernel-yamls: $(addprefix update-kernel-hash-yaml-,$(KERNELS)) update-kernel-semver-yaml-$(KERNEL_HIGHEST);

# Target for kernel config
KCONFIG_TAG_EXTENSION=
ifneq (${KCONFIG_TAG},)
KCONFIG_TAG_EXTENSION=-${KCONFIG_TAG}
endif

kconfig:
	docker build --no-cache -f Dockerfile.kconfig \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		--platform $(BUILD_PLATFORM) \
		-t linuxkit/kconfig:$(ARCH)${KCONFIG_TAG_EXTENSION} .

kconfigx:
ifeq (${KCONFIG_TAG},)
	docker buildx build --no-cache -f Dockerfile.kconfigx \
		--platform $(BUILD_PLATFORM) \
		--output . \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfigx:$(ARCH)  .
	cp linux_arm64/config-${KERNEL_VERSIONS}-arm64 config-${KERNEL_SERIES}-aarch64
	cp linux_amd64/config-${KERNEL_VERSIONS}-amd64 config-${KERNEL_SERIES}-x86_64
	cp linux_amd64/config-${KERNEL_VERSIONS}-riscv64 config-${KERNEL_SERIES}-riscv64
else
	docker buildx build --no-cache -f Dockerfile.kconfigx \
		--platform $(BUILD_PLATFORM) --push \
		--output . \
		--build-arg KERNEL_VERSIONS="$(KERNEL_VERSIONS)" \
		--build-arg BUILD_IMAGE=$(IMAGE_BUILDER) \
		-t linuxkit/kconfigx:$(ARCH)${KCONFIG_TAG_EXTENSION}  .
endif
