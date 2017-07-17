default: push

COMMON_IMAGES := \
  kube-proxy-amd64\:v1.6.1 \
  k8s-dns-sidecar-amd64\:1.14.1 \
  k8s-dns-kube-dns-amd64\:1.14.1 \
  k8s-dns-dnsmasq-nanny-amd64\:1.14.1 \
  pause-amd64\:3.0

CONTROL_PLANE_IMAGES := \
  kube-apiserver-amd64\:v1.6.1 \
  kube-controller-manager-amd64\:v1.6.1 \
  kube-scheduler-amd64\:v1.6.1 \
  etcd-amd64\:3.0.17

dl/%.tar:
	mkdir -p $(dir $@)
	docker image pull gcr.io/google_containers/$(shell basename $@ .tar)
	docker image save -o $@ gcr.io/google_containers/$(shell basename $@ .tar)

%-pkg:
	set -e ; \
	builddir=$$(mktemp -d $(CACHE).XXXXXX) ; \
	trap 'rm -rf $${builddir}' EXIT ; \
	ln $(IMAGES) $${builddir} ; \
	$(MAKE) -f Makefile.pkg BUILDDIR=$${builddir} CACHE=$(CACHE) $*

.PHONY: tag-common push-common
tag-common push-common: %-common: $(patsubst %,dl/%.tar,$(COMMON_IMAGES))
	$(MAKE) CACHE=common IMAGES="$^" $*-pkg

.PHONY: tag-control-plane push-control-plane
tag-control-plane push-control-plane: %-control-plane: $(patsubst %,dl/%.tar,$(CONTROL_PLANE_IMAGES))
	$(MAKE) CACHE=control-plane IMAGES="$^" $*-pkg

.PHONY: tag push
tag: tag-common tag-control-plane
push: push-common push-control-plane

.PHONY: dl
dl: $(patsubst %,dl/%.tar,$(COMMON_IMAGES) $(CONTROL_PLANE_IMAGES))

.PHONY: clean
clean:
	rm -rf dl
