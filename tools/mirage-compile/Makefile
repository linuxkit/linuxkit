.PHONY: tag push

BASE=ocaml/opam:alpine-3.5_ocaml-4.04.0
IMAGE=mirage-compile

default: push

hash: Dockerfile compile.sh
	docker pull $(BASE)
	tar cf - $^ | docker build -t $(IMAGE):build -
	docker run --rm --entrypoint=/bin/sh $(IMAGE):build -c \
	  '{ dpkg-query -W; \
             opam list; \
	     cat /usr/bin/compile.sh; \
           } | sha1sum' | sed 's/ .*//' > hash

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
	rm -f hash $(HASHES)

.DELETE_ON_ERROR:
