.PHONY: tag push

default: push

hash:
	docker pull alpine:3.4
	tar cf - Dockerfile | docker build --no-cache -t mobylinux/alpine-base:build -
	rm -f hash
	docker run mobylinux/alpine-base:build sha1sum /lib/apk/db/installed | sed 's/ .*//' > hash

push: hash
	docker pull mobylinux/alpine-base:$(shell cat hash) || \
		(docker tag mobylinux/alpine-base:build mobylinux/alpine-base:latest && \
		 docker tag mobylinux/alpine-base:build mobylinux/alpine-base:$(shell cat hash) && \
		 docker push mobylinux/alpine-base:$(shell cat hash) && \
		 docker push mobylinux/alpine-base:latest)
	rm -f hash

tag: hash
	docker pull mobylinux/alpine-base:$(shell cat hash) || \
		(docker tag mobylinux/alpine-base:build mobylinux/alpine-base:latest && \
		 docker tag mobylinux/alpine-base:build mobylinux/alpine-base:$(shell cat hash))
	rm -f hash

clean:
	rm -f hash
