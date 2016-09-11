# no easy hashing scheme for Debian, will switch to Alpine soon so temporarily use sha256 
.PHONY: push

BASE=debian:jessie
IMAGE=debian-build-kernel

push:
	docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t mobylinux/$(IMAGE):latest -
	docker push mobylinux/$(IMAGE):latest
