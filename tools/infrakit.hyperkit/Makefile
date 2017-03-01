.PHONY: build-in-container build-local
DEPS:=$(wildcard cmd/*.go) Dockerfile.build Makefile

# Version/Revision information
VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty='.m' --always || echo "Unknown")
REVISION ?= $(shell git rev-list -1 HEAD || echo "Unknown")

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t infrakit-hyperkit-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}:/go/src/github.com/docker/infrakit.hyperkit \
		infrakit-hyperkit-build	build-local VERSION=${VERSION} REVISION=${REVISION}

build-local: build/infrakit-instance-hyperkit

build/infrakit-instance-hyperkit: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ \
		--ldflags '-extldflags "-fno-PIC" -X github.com/docker/diagkit/pkg/gather.Version=$(VERSION) -X github.com/docker/diagkit/pkg/gather.Revision=$(REVISION)' \
		cmd/main.go cmd/instance.go
clean:
	rm -rf build

fmt:
	@echo "+ $@"
	@gofmt -s -l . 2>&1 | grep -v ^vendor/ | xargs gofmt -s -l -w

lint:
	@echo "+ $@"
	$(if $(shell which golint || echo ''), , \
		$(error Please install golint))
	@test -z "$$(golint ./... 2>&1 | grep -v ^vendor/ | grep -v mock/ | tee /dev/stderr)"

# this will blow away the vendored code and update it to latest
.PHONY: revendor revendor-local
revendor:
	@echo "+ $@"
	@rm -rf vendor.conf vendor
	@docker build -t infrakit-hyperkit-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}:/go/src/github.com/docker/infrakit.hyperkit \
		infrakit-hyperkit-build	revendor-local VERSION=${VERSION} REVISION=${REVISION}

revendor-local:
	@echo "+ $@"
	@go get github.com/LK4D4/vndr
	@vndr init
