DOCKER_VERSION=1.11.0-rc4
ARCH?=x86_64
OS?=Linux

all: bin

bin: docker.git
	(cd docker.git && git fetch origin --tags)
	cd docker.git && \
	git checkout v${DOCKER_VERSION} && \
	git branch -D v${DOCKER_VERSION}-patched; \
	git checkout -b v${DOCKER_VERSION}-patched && \
	(for i in $(shell ls -1 patches); do patch -p1 < ../patches/$${i}; git commit -a -s -m "applied $${i}"; done)
	BIND_DIR=$(shell pwd) make -C docker.git binary
	mkdir -p bin
	cp docker.git/bundles/${DOCKER_VERSION}/binary/* bin/
	rm -f bin/*.md5
	rm -f bin/*.sha256

docker.git:
	git clone git://github.com/docker/docker.git docker.git

clean:
	rm -rf bin
