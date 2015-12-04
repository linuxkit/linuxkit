DOCKER?=docker

.PHONY: build
build: VERSION
	$(DOCKER) build -t boot2docker:test .
	$(DOCKER) run -i boot2docker:test tar -C /tmp/iso/boot -c . | tar -xv

.PHONY: VERSION
VERSION:
	docker -v | cut -f 3 -d ' ' | cut -f 1 -d ',' > VERSION
