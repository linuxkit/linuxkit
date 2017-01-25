# Tag: 893c93bf54bc037f6952886330d5ba58746ace37
RIDDLER=mobylinux/riddler@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

# Tag: e79cbcc45b715cea7047a802944b478a7b5a906a
TEST_IMAGE=mobylinux/test@sha256:bc9403a9fc7aa5298f92c83d1e3423e804097f08dbba623e076c1728666f6b73

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
