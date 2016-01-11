all: docker

DOCKER_VERSION=1.9.1
ARCH=x86_64
OS=Linux

docker:
	curl -L -o docker https://get.docker.com/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}
	chmod +x docker

clean:
	rm -f docker
