BASE=ocaml/opam:alpine-3.5_ocaml-4.04.0
FILES=$(shell find . -name jbuild) \
	$(shell find sdk/ -name '*.ml') \
	$(shell find sdk/ -name '*.mli') \
	dhcp-client/bpf/dhcp.c dhcp-client/main.ml
IMAGE=dhcp-client
OBJS=obj/dhcp-client

MIRAGE_COMPILE=mobylinux/mirage-compile:f903b0e1b4328271364cc63f123ac49d56739cef@sha256:a54d9ca84d3f5998dba92ce83d60d49289cee8908a8b0f6ec280d30ab8edf46c
CALF_OBJS=obj/calf/dhcp-client-calf
CALF_FILES=dhcp-client/calf/config.ml dhcp-client/calf/unikernel.ml \
	   dhcp-client/calf/jbuild

default: push
	@

.build: Dockerfile.build $(FILES)
	docker build -t $(IMAGE):build -f Dockerfile.build -q . > .build || \
	 (rm -f $@ && exit 1)

.pkg: Dockerfile.pkg $(OBJS) $(CALF_OBJS) obj/config.json
	docker build --no-cache -t $(IMAGE):pkg -f Dockerfile.pkg -q . > .pkg || \
	  (rm -f $@ && exit 1)

.dev: Dockerfile.dev init-dev.sh
	docker build -t $(IMAGE):dev -f Dockerfile.dev -q . > .dev || \
	  (rm -f $@ && exit 1)

enter-pkg: .pkg
	docker run -v /bin/sh:/bin/sh -it --rm --entrypoint=/bin/sh $(shell cat .pkg)

enter-build: .build
	docker run -it --rm $(shell cat .build)

enter-dev: .dev
	docker run --privileged -it -v `pwd`:/src $(shell cat .dev)

#$(CALF_OBJS): $(CALF_FILES)
#	mkdir -p obj/bin
#	( cd obj && \
#	  tar -C ../dhcp-client/calf -cf - $(CALF_FILES:dhcp-client/calf/%=%) | \
#	    docker run --rm -i --log-driver=none $(MIRAGE_COMPILE) -o dhcp-client-calf | \
#	    tar xf - || exit 1) && \
#	touch $@

$(OBJS): .build $(FILES)
	mkdir -p obj/calf
	( cd obj && \
	  docker run --rm --net=none --log-driver=none -i $(IMAGE):build tar -cf - $(OBJS:obj/%=/%) | tar xf - ) && \
	touch $@

$(CALF_OBJS): .build $(CALF_FILES)
	mkdir -p obj/calf
	( cd obj && \
	  docker run --rm --net=none --log-driver=none -i $(IMAGE):build tar -cf - $(CALF_OBJS:obj/%=/%) | tar xf - ) && \
	touch $@

obj/config.json: dhcp-client/calf/config.json
	mkdir -p obj/calf
	cp $^ $@

hash: Makefile Dockerfile.build Dockerfile.pkg $(FILES) $(CALF_FILES) .build obj/config.json
	{ cat $^; \
	  docker run --rm --entrypoint sh $(IMAGE):build -c 'cat /lib/apk/db/installed'; \
	  docker run --rm --entrypoint sh $(IMAGE):build -c 'opam list'; } \
	| sha1sum | sed 's/ .*//' > $@

push: hash .pkg
	docker pull $(BASE)
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):pkg mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))

clean::
	rm -rf hash obj .build .pkg .dev
	(docker rmi -f $(IMAGE):build || echo ok)
	(docker rmi -f $(IMAGE):pkg || echo ok)
	(docker rmi -f $(IMAGE):dev || echo ok)

#### DEV

.PHONY: test

test:
	jbuilder runtest --dev

dev-clean:
	rm -rf _build dhcp-client/calf/_build

dev:
	jbuilder build dhcp-client/main.exe --dev
	jbuilder build dhcp-client/calf/unikernel.exe --dev
#	cd dhcp-client/calf && mirage configure && make

.DELETE_ON_ERROR:
