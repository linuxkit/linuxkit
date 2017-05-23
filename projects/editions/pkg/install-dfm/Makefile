default=tag
ORG?=linuxkit
IMAGE=install-dfm
DEPS=Dockerfile build/dfm/port build/dfm/docker

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

# Create empty dir for /port mount
build/dfm/port: Makefile
	mkdir -p build/dfm/port

# Configure initial docker dir, install memlogd plugin
build/dfm/docker: Makefile
	mkdir -p build/dfm/docker/plugins
	docker run --rm --privileged \
		-v $(PWD)/build/dfm/docker:/var/lib/docker \
		--name $(IMAGE)-dind \
		-d docker:17.05.0-ce-dind \
		--storage-driver=overlay2
	docker run --rm --link $(IMAGE)-dind:docker docker:edge plugin ls | grep -q memlogd-docker || ( \
		docker run --rm --link $(IMAGE)-dind:docker docker:edge plugin install cpuguy83/memlogd-docker --grant-all-permissions || ( \
			docker stop $(IMAGE)-dind ; exit 1 ) )
	docker stop $(IMAGE)-dind

tag: $(DEPS)
	tar --exclude=.gitignore -cf - $(DEPS) | docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) -

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)

clean:
	rm -rf build
