# Tag: 893c93bf54bc037f6952886330d5ba58746ace37
RIDDLER=mobylinux/riddler@sha256:3d4a61555110be4b6e8ff6bcdcf5f8aa24d64564eb4162ea4e580d8916d083cc

# Tag: c3312201a71982a820067a521e457a8c29aa7397
BINFMT_IMAGE=mobylinux/binfmt@sha256:2d08969710368376d927b0fc51796a2c30c67070421d4eb1973a08a83f98181f

default: container.tar

container.tar:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(RIDDLER) \
	   $(BINFMT_IMAGE) /containers/binfmt --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $(BINFMT_IMAGE) /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc >$@

clean:
	rm -f container.tar

.DELETE_ON_ERROR:
