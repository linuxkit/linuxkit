DOCKER_VERSION?=1.12.1-rc2
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=1

all: bin

TEST_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
DOCKER_HOST=$(shell [ "${DOCKER_EXPERIMENTAL}" -eq 1 ] && printf "experimental.docker.com" || printf "${TEST_HOST}")

bin:
	mkdir -p bin
	curl -sSL -o docker.tgz https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz
	tar xzf docker.tgz && mv docker/* bin
	rm -rf docker/ docker.tgz
	chmod +x bin/*

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin docker/ docker.tgz
