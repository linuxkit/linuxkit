RIDDLER=mobylinux/riddler:7d4545d8b8ac2700971a83f12a3446a76db28c14@sha256:11b7310df6482fc38aa52b419c2ef1065d7b9207c633d47554e13aa99f6c0b72

BINFMT_IMAGE=mobylinux/binfmt:a94e0587b702edaa95cc6f303464959d0eb2311c@sha256:432732b90cbe0498f5ca148d75b90bb1eabd8fbfe8c872df8b23906c225091b1

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	   $(BINFMT_IMAGE) /containers/binfmt --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $(BINFMT_IMAGE) /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
