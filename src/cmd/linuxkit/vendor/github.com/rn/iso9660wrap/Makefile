.PHONY: build-in-container build-local
DEPS:=$(wildcard *.go) Dockerfile.build Makefile

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t iso9660wrap-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}:/go/src/github.com/rneugeba/iso9660wrap \
		iso9660wrap-build build-local

build-local: build/iso9660wrap

build/iso9660wrap: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ \
		--ldflags '-extldflags "-fno-PIC"' \
		cmd/main.go
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
