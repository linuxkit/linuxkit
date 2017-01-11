# Tag d584b5be1bfb0dda11751706218e95ac0ddd0fa8
DOCKER_IMAGE=mobylinux/binfmt@sha256:45bab9a0203be3da86d4e878e7c7b9869c396d0cd8d3443de6484fd3eab61777

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
	../riddler.sh --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $$IMAGE /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc >$@

clean:
	rm -rf rootfs config.json
	docker rmi $(DOCKER_IMAGE)

.DELETE_ON_ERROR:
