# Tag: 6479aea36e0c3d177297cc936db5cbf93ece467c
TEST_IMAGE=mobylinux/test@sha256:5425a613bfbb9563d122c21a4a5377cc4cf836a5b201accfecf596a13c8dc607

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
