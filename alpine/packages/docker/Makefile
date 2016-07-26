DOCKER_VERSION=1.12.0-rc5
ARCH?=x86_64
OS?=Linux
EXPERIMENTAL?=1

all: bin

#DOCKER_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
DOCKER_HOST=experimental.docker.com

bin:
	mkdir -p bin
	curl -f -L -o docker.tgz https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz
	tar xzf docker.tgz && mv docker/* bin
	rm -rf docker/ docker.tgz
	chmod +x bin/*

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin docker/ docker.tgz
