RIDDLER=mobylinux/riddler:893c93bf54bc037f6952886330d5ba58746ace37@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

BINFMT_IMAGE=mobylinux/binfmt:e8e8e984e296148f1339c6cf335f55af2b9541a2@sha256:0c46639794bd6acaa71282dca282d8c113b1a7310b06be02cd850c07116c7d15

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	   $(BINFMT_IMAGE) /containers/binfmt --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $(BINFMT_IMAGE) /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
