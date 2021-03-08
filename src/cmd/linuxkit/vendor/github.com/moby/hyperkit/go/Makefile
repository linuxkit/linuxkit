.PHONY: build-in-container build-local
DEPS:=$(wildcard sample/*.go) $(wildcard *.go) Dockerfile.build Makefile

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t hyperkitgo-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}:/go/src/github.com/moby/hyperkit/go \
		hyperkitgo-build build-local

build-local: build/hyperkitgo

build/hyperkitgo: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		sample/main.go

# The next two targets make sure we can cross-compile for Linux and Windows.
# While it makes little sense to run the sample code, it makes it easier for
# consumers of this package if it cross compiles.
build/hyperkitgo_linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		sample/main.go
build/hyperkitgo.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		sample/main.go

.PHONY: clean fmt lint test
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

test:
	@echo "+ $@"
# We pass the module path (not just `.`) because in the CI we play
# tricks with symlinks that confuse go test: when we enter `go/`, we
# are no longer in the go tree, but back in ~/distiller/hyperkit/go
# where Circle CI forces our checkout.
	@go test -v $(TESTFLAGS) github.com/moby/hyperkit/go

# this will blow away the vendored code and update it to latest
.PHONY: vendor vendor-local
vendor:
	@echo "+ $@"
	@rm -rf vendor.conf vendor
	@docker build -t hyperkitgo-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}:/go/src/github.com/moby/hyperkit/go \
		hyperkitgo-build vendor-local VERSION=${VERSION} REVISION=${REVISION}

vendor-local:
	@echo "+ $@"
	@go get github.com/LK4D4/vndr
	@vndr init

.PHONY: ci setup
setup:
	go get github.com/golang/lint/golint

ci: setup build/hyperkitgo build/hyperkitgo_linux build/hyperkitgo.exe test
	test -z "$$(gofmt -s -l . 2>&1 | grep -v ^vendor/ | tee /dev/stderr)"
	test -z "$$(golint ./... 2>&1 | grep -v ^vendor/ | grep -v mock/ | tee /dev/stderr)"
