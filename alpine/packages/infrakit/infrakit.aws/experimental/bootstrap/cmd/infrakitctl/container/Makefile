# Makefile for building the watcher controller
#
# We use a container to compile the binary, and in
# the container we use govendor to fetch all the versioned
# packages and dependencies.
# If you need to change the version of a package, update
# the vendor/vendor.json and the build container will
# automatically pull in the versioned dependencies without
# changing your local workspace.

# Build version metadata
VERSION?=0.0-snapshot
REVISION=$(shell git rev-list -1 HEAD)

DOCKER_REPO_OWNER?=infrakit
DOCKER_REPO?=${DOCKER_REPO_OWNER}/infrakitctl
DOCKER_TAG?=dev
DOCKER_IMAGE=${DOCKER_REPO}:${DOCKER_TAG}

.PHONY: clean all container
.DEFAULT: all
all: container

clean:
	@echo "+ $@"
	rm -rf bin
	@# Remove local images
	docker rmi -f ${DOCKER_IMAGE} >/dev/null 2>&1 || true
	docker rmi -f ${DOCKER_REPO}:latest >/dev/null 2>&1 || true

container: clean
	@echo "+ $@"
	mkdir -p bin
	cd ../../.. && LDFLAGS="-X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
	  ./hack/build-in-docker 'cmd/infrakitctl/*.go' cmd/infrakitctl/container/bin/matchetectl
	docker build -t $(DOCKER_IMAGE) -t $(DOCKER_REPO):latest -t $(DOCKER_REPO):$(VERSION) .

ifeq (${DOCKER_PUSH},true)
	docker push ${DOCKER_IMAGE}
ifeq (${DOCKER_TAG_LATEST},true)
	docker push ${DOCKER_REPO}:latest
endif
endif
