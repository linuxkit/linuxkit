default: test.img

RIDDLER=mobylinux/riddler:7d4545d8b8ac2700971a83f12a3446a76db28c14@sha256:11b7310df6482fc38aa52b419c2ef1065d7b9207c633d47554e13aa99f6c0b72

TEST_IMAGE=mobylinux/test:725422886c3ea8fca2e0f4a8df5c6442cb1e937b@sha256:a077b8f84a51a4f4271b62adaa44985a4feff3d0a82b0d5fa8a46b45378428bc

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	  $(TEST_IMAGE) /test --cap-drop all --cap-add SYS_ADMIN -e HOME=/tmp \
	  -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock:ro \
	  -v /usr/bin/docker:/usr/bin/docker:ro \
	  -v /etc/resolv.conf:/etc/resolv.conf:ro \
	  --net host --read-only $(TEST_IMAGE) /bin/sh /bin/test.sh >$@

TAR2INITRD_IMAGE=mobylinux/tar2initrd:d5711601eb5b89de0f052d87365e18388ff3f1b5@sha256:58d377e65845f91400e173ce9fca93462f2f237947eef2b0d2c17bb4f2da5ee8

test.img: container.tar
	cat $^ | docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TAR2INITRD_IMAGE) > $@

clean:
	rm -f container.tar test.img

.DELETE_ON_ERROR:
