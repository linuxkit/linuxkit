default: push

include versions.mk

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

.PHONY: build-common forcebuild-common push-common forcepush-common show-tag-common
build-common forcebuild-common push-common forcepush-common show-tag-common: %-common: $(patsubst %,dl/%.tar,$(COMMON_IMAGES))
	@$(MAKE) CACHE=common IMAGES="$^" $*-pkg

.PHONY: build-control-plane forcebuild-control-plane push-control-plane forcepush-control-plane show-tag-control-plane
build-control-plane forcebuild-control-plane push-control-plane forcepush-control-plane show-tag-control-plane: %-control-plane: $(patsubst %,dl/%.tar,$(CONTROL_PLANE_IMAGES))
	@$(MAKE) CACHE=control-plane IMAGES="$^" $*-pkg

.PHONY: build forcebuild push forcepush show-tags
build: build-common build-control-plane
forcebuild: forcebuild-common forcebuild-control-plane
push: push-common push-control-plane
forcepush: forcepush-common forcepush-control-plane
show-tags: show-tag-common show-tag-control-plane

.PHONY: dl
dl: $(patsubst %,dl/%.tar,$(COMMON_IMAGES) $(CONTROL_PLANE_IMAGES))

.PHONY: clean
clean:
	rm -rf dl

.PHONY: refresh
refresh:
	./mkversions > versions.mk
