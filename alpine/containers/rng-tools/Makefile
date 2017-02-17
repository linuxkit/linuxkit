RIDDLER=mobylinux/riddler:7d4545d8b8ac2700971a83f12a3446a76db28c14@sha256:11b7310df6482fc38aa52b419c2ef1065d7b9207c633d47554e13aa99f6c0b72

RNGD_IMAGE=mobylinux/rngd:3dad6dd43270fa632ac031e99d1947f20b22eec9@sha256:1c93c1db7196f6f71f8e300bc1d15f0376dd18e8891c8789d77c8ff19f3a9a92

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	  $(RNGD_IMAGE) /containers/rngd --cap-drop all --cap-add SYS_ADMIN --read-only --oom-score-adj -800 $(RNGD_IMAGE) /bin/tini /usr/sbin/rngd -f >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
