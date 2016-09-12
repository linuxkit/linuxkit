# no easy hashing scheme for Debian, will switch to Alpine soon so temporarily use sha256 
.PHONY: push

BASE=debian:jessie
IMAGE=debian-build-kernel

TAG := $(shell cat /dev/urandom | od -N6 -t x2 | head -n1 | cut -b9- | sed 's/ //g')

push:
	docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t mobylinux/$(IMAGE):$(TAG) -
	docker tag mobylinux/$(IMAGE):$(TAG) mobylinux/$(IMAGE):latest
	docker push mobylinux/$(IMAGE):$(TAG)
	docker push mobylinux/$(IMAGE):latest
