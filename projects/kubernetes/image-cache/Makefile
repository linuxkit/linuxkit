default: push

COMMON_IMAGES := \
  kube-proxy-amd64\:v1.7.2@sha256\:d455480e81d60e0eff3415675278fe3daec6f56c79cd5b33a9b76548d8ab4365 \
  k8s-dns-sidecar-amd64\:1.14.4@sha256\:97074c951046e37d3cbb98b82ae85ed15704a290cce66a8314e7f846404edde9 \
  k8s-dns-kube-dns-amd64\:1.14.4@sha256\:40790881bbe9ef4ae4ff7fe8b892498eecb7fe6dcc22661402f271e03f7de344 \
  k8s-dns-dnsmasq-nanny-amd64\:1.14.4@sha256\:aeeb994acbc505eabc7415187cd9edb38cbb5364dc1c2fc748154576464b3dc2 \
  pause-amd64\:3.0@sha256\:163ac025575b775d1c0f9bf0bdd0f086883171eb475b5068e7defa4ca9e76516

CONTROL_PLANE_IMAGES := \
  kube-apiserver-amd64\:v1.7.2@sha256\:a9ccc205760319696d2ef0641de4478ee90fb0b75fbe6c09b1d64058c8819f97 \
  kube-controller-manager-amd64\:v1.7.2@sha256\:2b268ab9017fadb006ee994f48b7222375fe860dc7bd14bf501b98f0ddc2961b \
  kube-scheduler-amd64\:v1.7.2@sha256\:b2e897138449e7a00508dc589b1d4b71e56498a4d949ff30eb07b1e9d665e439 \
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
