# determine the version we save in the build binary
# we always include the git commit.
# the version is the current semver if it this commit matches the tag,
# else it is the following: <tag>-<commits since tag>-<short commit hash>
# if the git tree is dirty, append "-dirty"
# most recent commit
GIT_COMMIT=$(shell git rev-list -1 HEAD)
# whether or not it is dirty, i.e. has uncommitted changes
GIT_DIRTY=$(shell git update-index -q --refresh && git diff-index --quiet HEAD -- . || echo "-dirty")
# most recent tag, might or might not point to GIT_COMMIT
GIT_TAG=$(shell git describe --tags --match="v*")
# include the possible "-dirty" suffix
VERSION=$(GIT_TAG)$(GIT_DIRTY)

report:
	@echo "VERSION: $(VERSION)"
	@echo "GIT_COMMIT: $(GIT_COMMIT)"
	@echo "GIT_DIRTY: $(GIT_DIRTY)"
	@echo "GIT_TAG: $(GIT_TAG)"
	@echo "VERSION: $(VERSION)"

GO_COMPILE?=linuxkit/go-compile:985a9db72a7e6941de5e1eb71c2b41b76bf0556f
export GO_FLAGS=-mod=vendor

ifeq ($(OS),Windows_NT)
LINUXKIT?=bin/linuxkit.exe
GOOS?=windows
else
LINUXKIT?=bin/linuxkit
GOOS?=$(shell uname -s | tr '[:upper:]' '[:lower:]')
endif
GOARCH?=amd64
ifneq ($(GOOS),linux)
CROSS+=-e GOOS=$(GOOS)
endif
ifneq ($(GOARCH),amd64)
CROSS+=-e GOARCH=$(GOARCH)
endif

.DELETE_ON_ERROR:

.PHONY: default all
default: $(LINUXKIT)
all: default

LINUXKIT_DEPS=$(wildcard *.go) $(wildcard */*.go) Makefile


ifeq ($(GOOS),darwin)
$(LINUXKIT): local-build sign | bin
else
$(LINUXKIT): tmp_linuxkit_bin.tar | bin
	tar -C $(dir $(LINUXKIT)) -xf $<
	rm $<
	touch $@
endif

tmp_linuxkit_bin.tar: $(LINUXKIT_DEPS)
	tar cf - -C . . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/linuxkit/linuxkit/src/cmd/linuxkit --ldflags "-X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.GitCommit=$(GIT_COMMIT) -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.Version=$(VERSION)" -o $(notdir $(LINUXKIT)) > $@

.PHONY: test-cross
test-cross:
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=darwin tmp_linuxkit_bin.tar
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=windows tmp_linuxkit_bin.tar
	$(MAKE) clean
	$(MAKE) -j 3 GOOS=linux tmp_linuxkit_bin.tar
	$(MAKE) clean

LOCAL_LDFLAGS += -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.GitCommit=$(GIT_COMMIT) -X github.com/linuxkit/linuxkit/src/cmd/linuxkit/version.Version=$(VERSION)

STATIC?=1
CGO_ENABLED?=1
BUILD_FLAGS=
PIE?=0

ifeq ($(STATIC), 1)
	CGO_ENABLED=0
endif

# darwin needs CGO to build for virtualization framework
# without CGO attempt to use virtualization framework will ends with error
ifeq ($(GOOS), darwin)
	CGO_ENABLED=1
endif

ifeq ($(PIE), 1)
	CGO_ENABLED=0
	BUILD_FLAGS+= --buildmode pie
	LOCAL_LDFLAGS+= -linkmode=external -s -w -extldflags \"-static-pie\"
endif

LOCAL_TARGET ?= $(LINUXKIT)

.PHONY: local-check local-build local-test local
local-check: $(LINUXKIT_DEPS)
	@echo gofmt... && o=$$(gofmt -s -l $(filter %.go,$(LINUXKIT_DEPS))) && if [ -n "$$o" ] ; then echo $$o ; exit 1 ; fi
	@echo govet... && go vet -printf=false ./...
	@echo golangci-lint... && golangci-lint run ./...
	@echo ineffassign... && ineffassign ./...

local-build:
	@mkdir -p $(dir LOCAL_TARGET)
	CGO_ENABLED=$(CGO_ENABLED) go build -o $(LOCAL_TARGET) $(BUILD_FLAGS) --ldflags "$(LOCAL_LDFLAGS)"

local-test: $(LINUXKIT_DEPS)
	go test -mod=vendor ./...

local: local-check local-build local-test

bin:
	mkdir -p $@

sign:
	codesign --entitlements linuxkit.entitlements --force -s - $(LOCAL_TARGET)

install:
	cp -R ./bin/* $(PREFIX)/bin
	
.PHONY: clean
clean:
	rm -rf bin *.log *-kernel *-cmdline *-state *.img *.iso *.gz *.qcow2 *.vhd *.vmx *.vmdk *.tar *.raw
