.PHONY: default pr all

default: check-deps test 
pr: check-deps test-pr
# TODO: all should point to test-all once ltp is no longer required
all: check-deps test-pr ltp

MOBY:=$(shell command -v moby 2> /dev/null)
LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)
RTF:=$(shell command -v rtf 2> /dev/null)

.PHONY: check-deps
check-deps:
ifndef MOBY
	$(error "moby is not available. please install it.")
endif
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif
ifndef RTF
	$(error "rtf is not available. please install it!")
endif


# TODO: Remove this section once we no longer depend on this in CI
### -------
define check_test_log
	@cat $1 |grep -q 'test suite PASSED'
endef

.PHONY: ltp
ltp: export CLOUDSDK_IMAGE_NAME?=test-ltp
ltp: $(LINUXKIT) test-ltp.img.tar.gz
	$(MOBY) build --pull cases/020_stress/000_ltp/test-ltp.yml
	$(LINUXKIT) push gcp test-ltp.img.tar.gz
	$(LINUXKIT) run gcp -skip-cleanup -machine n1-highcpu-4 $(CLOUDSDK_IMAGE_NAME) | tee test-ltp.log
	$(call check_test_log, test-ltp.log)
### ------

test:
	@rtf -l build -x run

test-pr:
	@rtf -vvv -l build -x run

test-all:
	@rtf -vvv -x -l build,slow run
