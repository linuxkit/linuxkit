.PHONY: tag push

default: push

hash:
	docker pull alpine:3.4
	tar cf - Dockerfile | docker build --no-cache -t justincormack/moby-alpine-base:build -
	rm -f hash
	docker run justincormack/moby-alpine-base:build sha1sum /lib/apk/db/installed | sed 's/ .*//' > hash

push: hash
	docker pull justincormack/moby-alpine-base:$(shell cat hash) || \
		(docker tag justincormack/moby-alpine-base:build justincormack/moby-alpine-base:latest && \
		 docker tag justincormack/moby-alpine-base:build justincormack/moby-alpine-base:$(shell cat hash) && \
		 docker push justincormack/moby-alpine-base:$(shell cat hash) && \
		 docker push justincormack/moby-alpine-base:latest)
	rm -f hash

tag: hash
	docker pull justincormack/moby-alpine-base:$(shell cat hash) || \
		(docker tag justincormack/moby-alpine-base:build justincormack/moby-alpine-base:latest && \
		 docker tag justincormack/moby-alpine-base:build justincormack/moby-alpine-base:$(shell cat hash))
	rm -f hash

clean:
	rm -f hash
