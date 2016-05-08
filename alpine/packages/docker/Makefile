DOCKER_VERSION=1.12.0-dev
ARCH?=x86_64
OS?=Linux

all: bin

bin: docker.git
	(cd docker.git && git fetch origin --tags && git checkout 1.12-integration)
	BIND_DIR=$(shell pwd) make -C docker.git binary
	mkdir -p bin
	cp docker.git/bundles/${DOCKER_VERSION}/binary-client/* bin/
	cp docker.git/bundles/${DOCKER_VERSION}/binary-daemon/* bin/
	rm -f bin/*.md5
	rm -f bin/*.sha256
	rm -f bin/docker-${DOCKER_VERSION}

docker.git:
	git clone https://github.com/docker/docker-1.12-integration.git docker.git

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin docker.git
