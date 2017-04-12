.PHONY: default all
default: bin/moby 
all: default

VERSION="0.0" # dummy for now
GIT_COMMIT=$(shell git rev-list -1 HEAD)

GO_COMPILE=mobylinux/go-compile:90607983001c2789911afabf420394d51f78ced8@sha256:188beb574d4702a92fa3396a57cabaade28003c82f9413c3121a370ff8becea4

MOBY?=bin/moby
GOOS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH=amd64
ifneq ($(GOOS),linux)
CROSS=-e GOOS=$(GOOS) -e GOARCH=$(GOARCH)
endif
ifeq ($(GOOS),darwin)
default: bin/infrakit-instance-hyperkit
endif

MOBY_DEPS=$(wildcard src/cmd/moby/*.go) Makefile vendor.conf
MOBY_DEPS+=$(wildcard src/initrd/*.go) $(wildcard src/pad4/*.go)
bin/moby: $(MOBY_DEPS) | bin
	tar cf - vendor src/initrd src/pad4 -C src/cmd/moby . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby --ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" -o $@ | tar xf -
	touch $@

INFRAKIT_DEPS=$(wildcard src/cmd/infrakit-instance-hyperkit/*.go) Makefile vendor.conf
bin/infrakit-instance-hyperkit: $(INFRAKIT_DEPS) | bin
	tar cf - vendor -C src/cmd/infrakit-instance-hyperkit . | docker run --rm --net=none --log-driver=none -i $(CROSS) $(GO_COMPILE) --package github.com/docker/moby -o $@ | tar xf -
	touch $@

bin:
	mkdir -p $@

.PHONY: test
test: $(MOBY)
	@MOBY=$$(if [ "$(MOBY)" = "bin/moby" ]; then echo $(PWD)/$(MOBY); else echo $(MOBY); fi) \
	     $(MAKE) -C test

.PHONY: ci ci-tag ci-pr
ci:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

ci-tag:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

ci-pr:
	$(MAKE) clean
	$(MAKE)
	$(MAKE) test

.PHONY: clean
clean:
	rm -rf bin *.log *-bzImage *-cmdline *.img *.iso *.tar.gz *.qcow2 *.vhd
	$(MAKE) -C test clean
