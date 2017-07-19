default: push

COMMON_IMAGES := \
  kube-proxy-amd64\:v1.6.1@sha256\:243f2120171330a26c2418a4367fb0f3cc3e92683b00d16e3cf8c7f92e25bf14 \
  k8s-dns-sidecar-amd64\:1.14.1@sha256\:d33a91a5d65c223f410891001cd379ac734d036429e033865d700a4176e944b0 \
  k8s-dns-kube-dns-amd64\:1.14.1@sha256\:33914315e600dfb756e550828307dfa2b21fb6db24fe3fe495e33d1022f9245d \
  k8s-dns-dnsmasq-nanny-amd64\:1.14.1@sha256\:89c9a1d3cfbf370a9c1a949f39f92c1dc2dbe8c3e6cc1802b7f2b48e4dfe9a9e \
  pause-amd64\:3.0@sha256\:163ac025575b775d1c0f9bf0bdd0f086883171eb475b5068e7defa4ca9e76516

CONTROL_PLANE_IMAGES := \
  kube-apiserver-amd64\:v1.6.1@sha256\:d4387dff51b1f9c94cd1cfac3a4694347970b90e911159ac6fe2d090c96a6184 \
  kube-controller-manager-amd64\:v1.6.1@sha256\:4bb17ede2e012898169d988facd08d5039d2dcb31532661d4dcdeb161d097d69 \
  kube-scheduler-amd64\:v1.6.1@sha256\:d3e661bf7bcfb10753e32e1a41615e60fbcddff63232f914e9326a2d1665ce33 \
  etcd-amd64\:3.0.17@sha256\:d83d3545e06fb035db8512e33bd44afb55dea007a3abd7b17742d3ac6d235940

dl/%.tar:
	mkdir -p $(dir $@)
	docker image pull gcr.io/google_containers/$(shell basename $@ .tar)
	docker image save -o $@ gcr.io/google_containers/$(shell basename $@ .tar)

%-pkg:
	@set -e ; \
	builddir=$$(mktemp -d $(CACHE).XXXXXX) ; \
	trap 'rm -rf $${builddir}' EXIT ; \
	ln $(IMAGES) $${builddir} ; \
	$(MAKE) -f Makefile.pkg BUILDDIR=$${builddir} CACHE=$(CACHE) $*

.PHONY: tag-common push-common show-tag-common
tag-common push-common show-tag-common: %-common: $(patsubst %,dl/%.tar,$(COMMON_IMAGES))
	@$(MAKE) CACHE=common IMAGES="$^" $*-pkg

.PHONY: tag-control-plane push-control-plane show-tag-control-plane
tag-control-plane push-control-plane show-tag-control-plane: %-control-plane: $(patsubst %,dl/%.tar,$(CONTROL_PLANE_IMAGES))
	@$(MAKE) CACHE=control-plane IMAGES="$^" $*-pkg

.PHONY: tag push show-tags
tag: tag-common tag-control-plane
push: push-common push-control-plane
show-tags: show-tag-common show-tag-control-plane

.PHONY: dl
dl: $(patsubst %,dl/%.tar,$(COMMON_IMAGES) $(CONTROL_PLANE_IMAGES))

.PHONY: clean
clean:
	rm -rf dl
