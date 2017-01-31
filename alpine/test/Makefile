RIDDLER=mobylinux/riddler:893c93bf54bc037f6952886330d5ba58746ace37@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

TEST_IMAGE=mobylinux/test:fc8e36cbd58b5e9078ae6aa57a348ca2fbec76dc@sha256:7155b16109270dc2d950f63279e5a682b2793b83bb26ff70c4783e6d723db5ba

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	  $(TEST_IMAGE) /test --cap-drop all --cap-add SYS_ADMIN -e HOME=/tmp \
	  -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock:ro \
	  -v /usr/bin/docker:/usr/bin/docker:ro \
	  -v /etc/resolv.conf:/etc/resolv.conf:ro \
	  --net host --read-only $(TEST_IMAGE) /bin/sh /bin/test.sh >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
