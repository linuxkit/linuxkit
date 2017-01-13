# Tag 48a34c44de78a7607b8b03e54efffc8ede461342
MKSH_IMAGE=mobylinux/mksh@sha256:b9ca6f8ec3d7855db91162555128429030e0ebc950881c1f09015c40d9e875d2
MKSH_FILE=mksh
MKSH_PATH=/bin

# Tag: 41e4b91c9a619e46f76ce2d024067c09b62f07b4
CACERT_IMAGE=mobylinux/ca-certificates@sha256:6ca2dca9cfb8534a55f3a17f8797943527db5bbac08c98a5c9a4836250f4c548
CACERT_FILE=ca-certificates.crt
CACERT_PATH=/etc/ssl/certs

default: config.json

$(MKSH_FILE):
	docker run --rm --net=none $(MKSH_IMAGE) tar cf - -C $(MKSH_PATH) $(MKSH_FILE) | tar xf -

$(CACERT_FILE):
	docker run --rm --net=none $(CACERT_IMAGE) tar cf - -C $(CACERT_PATH) $(CACERT_FILE) | tar xf -

config.json: Dockerfile test.sh $(MKSH_FILE) $(CACERT_FILE)
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
	../containers/riddler.sh --cap-drop all --cap-add SYS_ADMIN -e HOME=/tmp -u 0:0 -v /var/tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock:ro -v /usr/bin/docker:/usr/bin/docker:ro -v /etc/resolv.conf:/etc/resolv.conf:ro --net host --read-only $$IMAGE /bin/test.sh >$@

clean:
	rm -rf rootfs config.json $(MKSH_FILE) $(CACERT_FILE)

.DELETE_ON_ERROR:
