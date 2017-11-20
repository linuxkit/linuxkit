.PHONY: tag push hash dynamic static cache-images run-static run-dynamic

ORG?=linuxkitprojects
IMAGE=compose
DEPS=image/Dockerfile docker-compose.yml image/waitfordocker.sh image/load-images-and-compose.sh
COMMON_IMAGES := \
	nginx:alpine \
	traefik


HASH?=$(shell git ls-tree HEAD -- ./image | awk '{print $$3}')

hash:
	@echo ${HASH}

tag: $(DEPS)
	docker build --squash --no-cache -t $(ORG)/$(IMAGE):$(HASH) image/

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)

dynamic:
	mkdir -p dist
	linuxkit build -name compose-dynamic -dir dist/ compose-dynamic.yml

static: cache-images
	mkdir -p dist
	linuxkit build -name compose-static -dir dist/ compose-static.yml

run-dynamic:
	linuxkit run dist/compose-dynamic

run-static:
	linuxkit run dist/compose-static

clean:
	rm -rf image-cache

image-cache/%.tar:
	mkdir -p $(dir $@)
	DOCKER_CONTENT_TRUST=1 docker image pull $(shell basename $@ .tar)
	docker image save -o $@ $(shell basename $@ .tar)

# use make here for each image rather than a single for loop so we can cache dependencies
cache-images:
	for image in $(COMMON_IMAGES) ; do \
		make "image-cache/$${image}.tar" ; \
	done
