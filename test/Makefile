.PHONY: default pr all

default: check-deps test 
pr: check-deps test-pr
# TODO: should have a separate target
all: check-deps test-pr ltp

LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)
RTF:=$(shell command -v rtf 2> /dev/null)

.PHONY: check-deps
check-deps:
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif
ifndef RTF
	$(error "rtf is not available. please install it!")
endif


# TODO: Remove this section once we no longer depend on this in CI
### -------
# Currently the linuxkit-ci runs GCP tests outside of rtf and expects some
# files in ../artifacts. This hacky target puts them there until
# the CI can use rtf for running GCP tests
gcp-hack: ../artifacts/test.img.tar.gz
../artifacts/test.img.tar.gz:
	rm -rf ../artifacts
	mkdir -p ../artifacts
	$(LINUXKIT) build -format gcp -pull -name ../artifacts/test hack/test.yml

define check_test_log
	@cat $1 |grep -q 'test suite PASSED'
endef

.PHONY: ltp
ltp: export CLOUDSDK_IMAGE_NAME?=test-ltp
ltp: $(LINUXKIT) test-ltp.img.tar.gz
	$(LINUXKIT) build -format gcp -pull hack/test-ltp.yml
	$(LINUXKIT) push gcp test-ltp.img.tar.gz
	$(LINUXKIT) run gcp -skip-cleanup -machine n1-highcpu-4 $(CLOUDSDK_IMAGE_NAME) | tee test-ltp.log
	$(call check_test_log, test-ltp.log)
### ------

test: gcp-hack
	@rtf -l build -v run -x

test-pr: gcp-hack
	@rtf -l build -v run -x
