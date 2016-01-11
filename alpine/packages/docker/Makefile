all: docker

DOCKER_VERSION=1.9.1
ARCH=x86_64
OS=Linux

docker:
	curl https://get.docker.com/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION} > docker
	chmod +x docker

clean:
	rm -f docker
