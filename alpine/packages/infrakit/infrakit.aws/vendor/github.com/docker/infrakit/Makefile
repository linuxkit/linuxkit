# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd -L)

# Used to populate version variable in main package.
VERSION?=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)
REVISION?=$(shell git rev-list -1 HEAD)

# Allow turning off function inlining and variable registerization
ifeq (${DISABLE_OPTIMIZATION},true)
	GO_GCFLAGS=-gcflags "-N -l"
	VERSION:="$(VERSION)-noopt"
endif

.PHONY: clean all fmt vet lint build test vendor-update containers
.DEFAULT: all
all: clean fmt vet lint build test binaries

ci: fmt vet lint coverage

AUTHORS: .mailmap .git/HEAD
	 git log --format='%aN <%aE>' | sort -fu > $@

# Package list
PKGS_AND_MOCKS := $(shell go list ./... | grep -v /vendor)
PKGS := $(shell echo $(PKGS_AND_MOCKS) | tr ' ' '\n' | grep -v /mock$)

# Current working environment.  Set these explicitly if you want to cross-compile
# in the build container (see the build-in-container target):

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
build-in-container: clean
	@echo "+ $@"
	@docker build -t infrakit-build -f ${CURDIR}/dockerfiles/Dockerfile.build .
	@docker run --rm \
		-e GOOS=${GOOS} -e GOARCCH=${GOARCH} \
		-v ${CURDIR}/build:/go/src/github.com/docker/infrakit/build \
		infrakit-build

vet:
	@echo "+ $@"
	@go vet $(PKGS)

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . 2>&1 | grep -v ^vendor/ | tee /dev/stderr)" || \
		(echo >&2 "+ please format Go code with 'gofmt -s', or use 'make fmt-save'" && false)

fmt-save:
	@echo "+ $@"
	@gofmt -s -l . 2>&1 | grep -v ^vendor/ | xargs gofmt -s -l -w

lint:
	@echo "+ $@"
	$(if $(shell which golint || echo ''), , \
		$(error Please install golint: `go get -u github.com/golang/lint/golint`))
	@test -z "$$(golint ./... 2>&1 | grep -v ^vendor/ | grep -v mock/ | tee /dev/stderr)"

build:
	@echo "+ $@"
	@go build ${GO_LDFLAGS} $(PKGS)

clean:
	@echo "+ $@"
	rm -rf build
	mkdir -p build

define build_binary
	go build -o build/$(1) \
	  -ldflags "-X github.com/docker/infrakit/cli.Version=$(VERSION) -X github.com/docker/infrakit/cli.Revision=$(REVISION)" $(2)
endef

binaries: clean build-binaries
build-binaries:
	@echo "+ $@"
ifneq (,$(findstring .m,$(VERSION)))
	@echo "\nWARNING - repository contains uncommitted changes, tagging binaries as dirty\n"
endif

	$(call build_binary,infrakit,github.com/docker/infrakit/cmd/cli)
	$(call build_binary,infrakit-group-default,github.com/docker/infrakit/cmd/group)
	$(call build_binary,infrakit-flavor-combo,github.com/docker/infrakit/example/flavor/combo)
	$(call build_binary,infrakit-flavor-swarm,github.com/docker/infrakit/example/flavor/swarm)
	$(call build_binary,infrakit-flavor-vanilla,github.com/docker/infrakit/example/flavor/vanilla)
	$(call build_binary,infrakit-flavor-zookeeper,github.com/docker/infrakit/example/flavor/zookeeper)
	$(call build_binary,infrakit-instance-file,github.com/docker/infrakit/example/instance/file)
	$(call build_binary,infrakit-instance-terraform,github.com/docker/infrakit/example/instance/terraform)
	$(call build_binary,infrakit-instance-vagrant,github.com/docker/infrakit/example/instance/vagrant)

install:
	@echo "+ $@"
	@go install ${GO_LDFLAGS} $(PKGS)

generate:
	@echo "+ $@"
	@go generate -x $(PKGS_AND_MOCKS)

test:
	@echo "+ $@"
	@go test -test.short -race -v $(PKGS)

coverage:
	@echo "+ $@"
	@for pkg in $(PKGS); do \
	  go test -test.short -race -coverprofile="../../../$$pkg/coverage.txt" $${pkg} || exit 1; \
	done

test-full:
	@echo "+ $@"
	@go test -race $(PKGS)

vendor-update:
	@echo "+ $@"
	@trash -u
