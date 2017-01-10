# Tag 7cb780fd6b60c089964e81efd6553853c491d59f
TINI_IMAGE=mobylinux/tini@sha256:7da8c5b371e0d7d3fb1778e96c0bc634e39ace7bf1e7a73bffbf1f8360127fdb
TINI_BINARY=tini

default: config.json

$(TINI_BINARY): Dockerfile
	docker run --rm --net=none $(TINI_IMAGE) tar cf - -C /bin $@ | tar xf -

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

config.json: Dockerfile fix-textrels-on-PIC-x86.patch sha256sums $(TINI_BINARY)
	mkdir -p rootfs
	BUILD=$$( tar cf - $^ | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	IMAGE=$$( docker run --rm --net=none $$BUILD | docker build -q - ) && \
	[ -n "$$IMAGE" ] && \
	echo "Built $$IMAGE" && \
	CONTAINER=$$( docker create $$IMAGE /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C rootfs $(EXCLUDE) && \
	docker rm $$CONTAINER && \
	../riddler.sh --cap-drop all --cap-add SYS_ADMIN --read-only $$IMAGE /bin/tini /usr/sbin/rngd -f >$@

clean:
	rm -rf rootfs config.json $(TINI_BINARY)

.DELETE_ON_ERROR:
