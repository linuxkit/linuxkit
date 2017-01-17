# Tag: 3dad6dd43270fa632ac031e99d1947f20b22eec9
RNGD_IMAGE=mobylinux/rngd@sha256:1c93c1db7196f6f71f8e300bc1d15f0376dd18e8891c8789d77c8ff19f3a9a92

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
