# Tag: c3312201a71982a820067a521e457a8c29aa7397
BINFMT_IMAGE=mobylinux/binfmt@sha256:2d08969710368376d927b0fc51796a2c30c67070421d4eb1973a08a83f98181f

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
