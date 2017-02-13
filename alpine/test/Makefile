default: test.img

RIDDLER=mobylinux/riddler:893c93bf54bc037f6952886330d5ba58746ace37@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

TEST_IMAGE=mobylinux/test:1a8c5de8ebfa27962f7623846e8538babdf48e76@sha256:7dada68bcb76e897a7a7aa2a28b6b1808b04e8eda7319e1da97bd1cff9952fee

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
