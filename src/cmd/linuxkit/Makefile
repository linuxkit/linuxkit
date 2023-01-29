VERSION?="v0.0-dev"
GIT_COMMIT=$(shell git rev-list -1 HEAD)
GO_COMPILE?=linuxkit/go-compile:c97703655e8510b7257ffc57f25e40337b0f0813
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
