# Tag: 893c93bf54bc037f6952886330d5ba58746ace37
RIDDLER=mobylinux/riddler@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

# Tag: 3dad6dd43270fa632ac031e99d1947f20b22eec9
RNGD_IMAGE=mobylinux/rngd@sha256:1c93c1db7196f6f71f8e300bc1d15f0376dd18e8891c8789d77c8ff19f3a9a92

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	  $(RNGD_IMAGE) /containers/rngd --cap-drop all --cap-add SYS_ADMIN --read-only --oom-score-adj -800 $(RNGD_IMAGE) /bin/tini /usr/sbin/rngd -f >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
