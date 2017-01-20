# Tag: e79cbcc45b715cea7047a802944b478a7b5a906a
TEST_IMAGE=mobylinux/test@sha256:bc9403a9fc7aa5298f92c83d1e3423e804097f08dbba623e076c1728666f6b73

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
