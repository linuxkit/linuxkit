BASE=ocaml/opam:alpine-3.5_ocaml-4.04.0

FILES=$(shell find . -name jbuild) \
	$(shell find sdk/ -name '*.ml') \
	$(shell find sdk/ -name '*.mli') \
	dhcp-client/bpf/dhcp.c dhcp-client/main.ml \
	dhcp-client-calf/unikernel.ml dhcp-client-calf/config.json

IMAGE=dhcp-client

default: push
	@

.build: Dockerfile $(FILES)
	docker build $(NO_CACHE) -t $(IMAGE) -f Dockerfile .
	docker build $(NO_CACHE) -t $(IMAGE) -f Dockerfile -q . > .build || \
	 (rm -f $@ && exit 1)

hash: Makefile Dockerfile $(FILES) .build
	{ cat $^; \
	  docker run --rm --entrypoint sh $(IMAGE) -c 'cat /lib/apk/db/installed'; \
	  docker run --rm --entrypoint sh $(IMAGE) -c 'opam list'; } \
	| sha1sum | sed 's/ .*//' > $@

tag: .build
	docker tag $(IMAGE) linuxkitprojects/$(IMAGE):$(shell cat hash)

push: hash .build
	docker pull $(BASE)
	docker pull linuxkitprojects/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE) linuxkitprojects/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkitprojects/$(IMAGE):$(shell cat hash))

clean::
	rm -rf hash .build
	(docker rmi -f $(IMAGE) || echo ok)

#### DEV

.PHONY: test

test:
	jbuilder runtest --dev

dev-clean:
	rm -rf _build

dev:
	jbuilder build dhcp-client/main.exe --dev
	jbuilder build dhcp-client-calf/unikernel.exe --dev

.DELETE_ON_ERROR:
