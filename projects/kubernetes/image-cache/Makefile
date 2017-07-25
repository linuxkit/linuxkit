default: push

COMMON_IMAGES := \
  kube-proxy-amd64\:v1.6.7@sha256\:652ca0ef7cdf05341fafb590ced1b737126641829c70f5d23f9b714bc61c8607 \
  k8s-dns-sidecar-amd64\:1.14.4@sha256\:d33a91a5d65c223f410891001cd379ac734d036429e033865d700a4176e944b0 \
  k8s-dns-kube-dns-amd64\:1.14.4@sha256\:33914315e600dfb756e550828307dfa2b21fb6db24fe3fe495e33d1022f9245d \
  k8s-dns-dnsmasq-nanny-amd64\:1.14.4@sha256\:89c9a1d3cfbf370a9c1a949f39f92c1dc2dbe8c3e6cc1802b7f2b48e4dfe9a9e \
  pause-amd64\:3.0@sha256\:163ac025575b775d1c0f9bf0bdd0f086883171eb475b5068e7defa4ca9e76516

CONTROL_PLANE_IMAGES := \
  kube-apiserver-amd64\:v1.6.7@sha256\:57e482529b95d32730d1bcd2e374199f27eab4abcf1ff49c5db2a7a7e2231cc8 \
  kube-controller-manager-amd64\:v1.6.7@sha256\:884f609895fa715d66806681a6bf6f9851a911202ad3484b768fead8b7c78b39 \
  kube-scheduler-amd64\:v1.6.7@sha256\:a1a498a0ca5ab23c228724d93c3f3d9457b31a046d9025471d98e1096422452c \
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
