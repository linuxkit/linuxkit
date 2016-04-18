DOCKER_VERSION=1.11.0
ARCH?=x86_64
OS?=Linux

all: bin

bin: docker.git
	(cd docker.git && git fetch origin --tags && git checkout moby)
	BIND_DIR=$(shell pwd) make -C docker.git binary
	mkdir -p bin
	cp docker.git/bundles/${DOCKER_VERSION}/binary/* bin/
	rm -f bin/*.md5
	rm -f bin/*.sha256

docker.git:
	git clone git://github.com/djs55/docker.git docker.git

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin
