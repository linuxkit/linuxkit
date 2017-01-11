# Tag c90d8f843ed63b44bf48b10e45400b96ce04ad20
DOCKER_IMAGE=mobylinux/rngd@sha256:12c32e791ed6f3bb7e444e8658f1d2ce5844e6aa80f23c0dbbcccf041a1cb32f

default: config.json

$(DOCKER_IMAGE):
	docker pull $(DOCKER_IMAGE)

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

config.json: $(DOCKER_IMAGE)
	mkdir -p rootfs
	CONTAINER=$$( docker create $(DOCKER_IMAGE) /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C rootfs $(EXCLUDE) && \
	docker rm $$CONTAINER && \
	../riddler.sh --cap-drop all --cap-add SYS_ADMIN --read-only $$IMAGE /bin/tini /usr/sbin/rngd -f >$@

clean:
	rm -rf rootfs config.json
	docker rmi $(DOCKER_IMAGE)

.DELETE_ON_ERROR:
