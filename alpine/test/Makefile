# Tag: a61cdb69d8b5c95c2766b2b856e4f331bd2a6d2d
TEST_IMAGE=mobylinux/test@sha256:1a2cb7acab25059532c5287c2cdde1a5cfc0fbb944593e02b85b5bb334a4187d

default: config.json

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

config.json:
	mkdir -p rootfs
	CONTAINER=$$( docker create $(TEST_IMAGE) /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C rootfs $(EXCLUDE) && \
	docker rm $$CONTAINER && \
	../containers/riddler.sh --cap-drop all --cap-add SYS_ADMIN -e HOME=/tmp -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock:ro -v /usr/bin/docker:/usr/bin/docker:ro -v /etc/resolv.conf:/etc/resolv.conf:ro --net host --read-only $(TEST_IMAGE) /bin/sh /bin/test.sh >$@

clean:
	rm -rf rootfs config.json

.DELETE_ON_ERROR:
