.PHONY: default pr all

default: check-deps test 
pr: check-deps test-pr
# TODO: should have a separate target
all: check-deps test-pr ltp

LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)
RTF:=$(shell command -v rtf 2> /dev/null)
# test suite to run, blank for all
TEST_SUITE ?=

.PHONY: check-deps
check-deps:
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif
ifndef RTF
	$(error "rtf is not available. please install it!")
endif


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

test:
	@rtf -l build -v run -x $(TEST_SUITE)

test-pr: test
