DOCKER?=docker

.PHONY: build
build: VERSION
	$(shell cd ../cmd/9pudc && GOOS=linux GOARCH=386 go build)
	cp ../cmd/9pudc/9pudc rootfs/rootfs/sbin/9pudc
	$(DOCKER) build -t boot2docker:test .
	$(DOCKER) run  boot2docker:test tar -C /tmp/iso/boot -c . | tar -xv

.PHONY: VERSION
VERSION:
	docker -v | cut -f 3 -d ' ' | cut -f 1 -d ',' > VERSION
