# Tag: 6fb2e0bd1844349222ad57af92b5c627fd73375a
RNGD_IMAGE=mobylinux/rngd@sha256:8370ecd6f5d2092b27b40c2dabe25a2cbeb6469dd6e973c27a5152af6ab8d12a

default: config.json

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

config.json:
	mkdir -p rootfs
	CONTAINER=$$( docker create $(RNGD_IMAGE) /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C rootfs $(EXCLUDE) && \
	docker rm $$CONTAINER && \
	../riddler.sh --cap-drop all --cap-add SYS_ADMIN --read-only $(RNGD_IMAGE) /bin/tini /usr/sbin/rngd -f >$@

clean:
	rm -rf rootfs config.json

.DELETE_ON_ERROR:
