# Tag: 6571d9d0c5c2592848f4f0202d1cd2c4466d9979
BINFMT_IMAGE=mobylinux/binfmt@sha256:6810b978316198cf4c507b901cfb676acd655955c380d2c98f23f7232ea6381f

default: config.json

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

config.json:
	mkdir -p rootfs
	CONTAINER=$$( docker create $(BINFMT_IMAGE) /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C rootfs $(EXCLUDE) && \
	docker rm $$CONTAINER && \
	../riddler.sh --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $(BINFMT_IMAGE) /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc >$@

clean:
	rm -rf rootfs config.json

.DELETE_ON_ERROR:
