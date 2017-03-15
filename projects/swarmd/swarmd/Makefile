.PHONY: tag push clean container
all: push

SHASUM=alpine:3.5
IMAGE=swarmd
DEPS=Dockerfile.build Makefile

# Include  Dockerfile.pkg here so hash works
swarmd.tag: $(DEPS) Dockerfile.pkg
	BUILD=$$(tar cf - $^ | docker build -f $< -q -) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

swarmd.tar: swarmd.tag
	docker run --rm --net=none --log-driver=none $(shell cat swarmd.tag) > $@

container: Dockerfile.pkg swarmd.tar
	tar cf - $^ | docker build --no-cache -f $< -t $(IMAGE):build -

hash: $(DEPS) Dockerfile.pkg
	find $^ -type f | xargs cat  | DOCKER_CONTENT_TRUST=1 docker run --rm -i $(SHASUM) sha1sum | sed 's/ .*//' > $@

push: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash
	rm -f swarmd.tag swarmd.tar

.DELETE_ON_ERROR:
