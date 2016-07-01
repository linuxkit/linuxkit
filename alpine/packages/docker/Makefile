DOCKER_VERSION=1.12-rc3-rc
ARCH?=x86_64
OS?=Linux
EXPERIMENTAL?=1

all: bin

bin: docker.git
	(cd docker.git && git fetch origin --tags && git checkout moby-$(DOCKER_VERSION))
	DOCKER_EXPERIMENTAL=$(EXPERIMENTAL) BIND_DIR=$(shell pwd) make -C docker.git binary
	mkdir -p bin
	cp docker.git/bundles/latest/binary-daemon/dockerd bin/
	cp docker.git/bundles/latest/binary-daemon/docker-runc bin/
	cp docker.git/bundles/latest/binary-daemon/docker-containerd bin/
	cp docker.git/bundles/latest/binary-daemon/docker-containerd-shim bin/
	cp docker.git/bundles/latest/binary-daemon/docker-containerd-ctr bin/
	cp docker.git/bundles/latest/binary-daemon/docker-proxy bin/
	cp docker.git/bundles/latest/binary-client/docker bin/

docker.git:
	git clone git://github.com/djs55/docker.git docker.git

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin docker.git
