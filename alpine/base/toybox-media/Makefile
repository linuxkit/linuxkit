.PHONY: tag push

# Tag: c544d1d4e0a5dd6af3e867c0fa19a628dc128e0c
BASE=mobylinux/alpine-build-toybox@sha256:3ef02ab3d9edebd9676b0178d729503d32f92230ef53132c26221d25911431d1
IMAGE=toybox-media

default: push

hash: 
	docker run --rm $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run --rm $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

push: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash

.DELETE_ON_ERROR:
