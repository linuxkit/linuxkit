all: docker

DOCKER_VERSION=1.9.1

docker:
	docker run docker:$(DOCKER_VERSION) cat /usr/local/bin/docker > docker

	chmod +x docker

clean:
	rm -f docker
